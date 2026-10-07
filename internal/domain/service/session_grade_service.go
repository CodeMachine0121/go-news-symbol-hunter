package service

import (
	"context"
	"errors"
	"fmt"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type SessionGradeService struct {
	symbolResolutionService    *SymbolResolutionService
	analystConsultationService *AnalystConsultationService
	trackedSymbolRepository    interfaces.ITrackedSymbolRepository
	sessionGradeRepository     interfaces.ISessionGradeRepository
	combinedGradeRepository    interfaces.ICombinedGradeRepository
	sessionRunRepository       interfaces.ISessionRunRepository
	clockProxy                 interfaces.IClockProxy
	sessionWeights             vo.SessionWeightsVo
}

func NewSessionGradeService(symbolResolutionService *SymbolResolutionService, analystConsultationService *AnalystConsultationService, trackedSymbolRepository interfaces.ITrackedSymbolRepository, sessionGradeRepository interfaces.ISessionGradeRepository, combinedGradeRepository interfaces.ICombinedGradeRepository, sessionRunRepository interfaces.ISessionRunRepository, clockProxy interfaces.IClockProxy, sessionWeights vo.SessionWeightsVo) *SessionGradeService {
	return &SessionGradeService{
		symbolResolutionService:    symbolResolutionService,
		analystConsultationService: analystConsultationService,
		trackedSymbolRepository:    trackedSymbolRepository,
		sessionGradeRepository:     sessionGradeRepository,
		combinedGradeRepository:    combinedGradeRepository,
		sessionRunRepository:       sessionRunRepository,
		clockProxy:                 clockProxy,
		sessionWeights:             sessionWeights,
	}
}

func (sessionGradeService *SessionGradeService) RunDueTradingSession(ctx context.Context) error {
	startedAt := sessionGradeService.clockProxy.Now()
	tradingDay := domains.NewTradingDayDomain(startedAt)
	session, isDue := tradingDay.DueSession()
	if !isDue {
		return nil
	}
	createdSessionRun := domains.NewStartedSessionRunDomain(tradingDay.Value(), session, startedAt).ToEntity()
	createError := sessionGradeService.sessionRunRepository.Create(ctx, &createdSessionRun)
	if errors.Is(createError, ErrSessionRunAlreadyExists) {
		return nil
	}
	if createError != nil {
		return fmt.Errorf("%w: %v", ErrSessionGradeStorageUnavailable, createError)
	}
	sessionRun := domains.NewSessionRunDomain(createdSessionRun)
	failureReason := ""
	if session == domains.TradingSessionPreMarket {
		if err := errors.Join(
			sessionGradeService.sessionGradeRepository.DeleteExceptTradingDay(ctx, tradingDay.Value()),
			sessionGradeService.combinedGradeRepository.DeleteExceptTradingDay(ctx, tradingDay.Value()),
		); err != nil {
			failureReason = FailureReasonStaleGradesNotPurged
		}
	}
	if failureReason == "" {
		trackedSymbols, findError := sessionGradeService.trackedSymbolRepository.FindTracking(ctx, vo.MarketCategoryTwStock)
		if findError != nil {
			failureReason = FailureReasonTrackedSymbolsUnavailable
		}
		for _, trackedSymbol := range trackedSymbols {
			sessionRun.RecordSymbolOutcome(sessionGradeService.gradeTrackedSymbol(ctx, trackedSymbol, tradingDay, session))
		}
	}
	if failureReason == "" {
		sessionRun.Succeed(sessionGradeService.clockProxy.Now())
	} else {
		sessionRun.Fail(failureReason, sessionGradeService.clockProxy.Now())
	}
	finishedSessionRun := sessionRun.ToEntity()
	if err := sessionGradeService.sessionRunRepository.Update(context.WithoutCancel(ctx), &finishedSessionRun); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionGradeStorageUnavailable, err)
	}
	return nil
}

// kept separate so each symbol's analysis deadline is released before the next symbol starts,
// instead of every deferred cancel piling up until the whole session finishes
func (sessionGradeService *SessionGradeService) gradeTrackedSymbol(ctx context.Context, trackedSymbol entities.TrackedSymbol, tradingDay domains.TradingDayDomain, session string) vo.SymbolGradingOutcomeVo {
	symbolGradingOutcome := vo.SymbolGradingOutcomeVo{Symbol: trackedSymbol.Symbol, Category: trackedSymbol.Category}
	analysisContext, cancelAnalysis := context.WithTimeout(ctx, domains.AnalysisTimeout)
	defer cancelAnalysis()
	resolvedSymbol, err := sessionGradeService.symbolResolutionService.ResolveSymbol(analysisContext, dto.ResolveSymbolDto{Symbol: trackedSymbol.Symbol, Category: trackedSymbol.Category})
	if err != nil {
		symbolGradingOutcome.FailureReason = err.Error()
		return symbolGradingOutcome
	}
	symbol := resolvedSymbol.Symbol
	analystConsultation := sessionGradeService.analystConsultationService.Consult(analysisContext, dto.ConsultAnalystDto{
		Symbol:              symbol.Value,
		Category:            symbol.Category.Value,
		SearchKeyword:       resolvedSymbol.SearchKeyword,
		NewsPublishedWindow: tradingDay.NewsPublishedWindow(session),
	})
	symbolGradingOutcome.Usage = analystConsultation.Usage()
	sessionGrade, isConcluded := analystConsultation.ToSessionGradeEntity(symbol, tradingDay.Value(), session, sessionGradeService.clockProxy.Now())
	if !isConcluded {
		symbolGradingOutcome.FailureReason = analystConsultation.FailureReason()
		return symbolGradingOutcome
	}
	storageContext := context.WithoutCancel(ctx)
	if err := sessionGradeService.sessionGradeRepository.Save(storageContext, &sessionGrade); err != nil {
		symbolGradingOutcome.FailureReason = FailureReasonSessionGradeNotSaved
		return symbolGradingOutcome
	}
	sessionGrades, err := sessionGradeService.sessionGradeRepository.FindByTradingDay(storageContext, symbol.Value, symbol.Category.Value, tradingDay.Value())
	if err != nil {
		symbolGradingOutcome.FailureReason = FailureReasonCombinedGradeNotSaved
		return symbolGradingOutcome
	}
	combinedGrade := domains.NewCombinedGradeDomain(sessionGrades, sessionGradeService.sessionWeights).ToEntity(symbol, tradingDay.Value(), sessionGradeService.clockProxy.Now())
	if err := sessionGradeService.combinedGradeRepository.Save(storageContext, &combinedGrade); err != nil {
		symbolGradingOutcome.FailureReason = FailureReasonCombinedGradeNotSaved
	}
	return symbolGradingOutcome
}

func (sessionGradeService *SessionGradeService) GetTrackedSymbolGrades(ctx context.Context, getTrackedSymbolGradesDto dto.GetTrackedSymbolGradesDto) ([]dto.TrackedSymbolGradeDto, error) {
	symbol := ""
	category := ""
	if getTrackedSymbolGradesDto.Symbol != "" || getTrackedSymbolGradesDto.Category != "" {
		if getTrackedSymbolGradesDto.Symbol == "" || getTrackedSymbolGradesDto.Category == "" {
			return nil, ErrSymbolAndCategoryRequiredTogether
		}
		marketCategory, err := vo.NewMarketCategoryVo(getTrackedSymbolGradesDto.Category)
		if err != nil {
			return nil, err
		}
		requestedSymbol, err := vo.NewSymbolVo(getTrackedSymbolGradesDto.Symbol, marketCategory)
		if err != nil {
			return nil, err
		}
		symbol = requestedSymbol.Value
		category = marketCategory.Value
	}
	combinedGrades, err := sessionGradeService.combinedGradeRepository.FindAll(ctx, symbol, category)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionGradeStorageUnavailable, err)
	}
	trackedSymbolGrades := make([]dto.TrackedSymbolGradeDto, 0, len(combinedGrades))
	for _, combinedGrade := range combinedGrades {
		sessionGrades, findError := sessionGradeService.sessionGradeRepository.FindByTradingDay(ctx, combinedGrade.Symbol, combinedGrade.Category, combinedGrade.TradingDay)
		if findError != nil {
			return nil, fmt.Errorf("%w: %v", ErrSessionGradeStorageUnavailable, findError)
		}
		trackedSymbolGrades = append(trackedSymbolGrades, domains.NewTrackedSymbolGradeDomain(combinedGrade, sessionGrades).ToDto())
	}
	return trackedSymbolGrades, nil
}

func (sessionGradeService *SessionGradeService) FailInterruptedSessionRuns(ctx context.Context) error {
	if err := sessionGradeService.sessionRunRepository.FailAllRunning(ctx, FailureReasonSessionRunInterrupted, sessionGradeService.clockProxy.Now()); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionGradeStorageUnavailable, err)
	}
	return nil
}
