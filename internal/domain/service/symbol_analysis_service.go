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

const maximumStartAttempts = 3

type SymbolAnalysisService struct {
	symbolResolutionService    *SymbolResolutionService
	analystConsultationService *AnalystConsultationService
	priceSnapshotService       *PriceSnapshotService
	analysisEventRepository    interfaces.IAnalysisEventRepository
	analysisResultRepository   interfaces.IAnalysisResultRepository
	clockProxy                 interfaces.IClockProxy
}

func NewSymbolAnalysisService(symbolResolutionService *SymbolResolutionService, analystConsultationService *AnalystConsultationService, priceSnapshotService *PriceSnapshotService, analysisEventRepository interfaces.IAnalysisEventRepository, analysisResultRepository interfaces.IAnalysisResultRepository, clockProxy interfaces.IClockProxy) *SymbolAnalysisService {
	return &SymbolAnalysisService{
		symbolResolutionService:    symbolResolutionService,
		analystConsultationService: analystConsultationService,
		priceSnapshotService:       priceSnapshotService,
		analysisEventRepository:    analysisEventRepository,
		analysisResultRepository:   analysisResultRepository,
		clockProxy:                 clockProxy,
	}
}

func (symbolAnalysisService *SymbolAnalysisService) StartSymbolAnalysis(ctx context.Context, startSymbolAnalysisDto dto.StartSymbolAnalysisDto) (dto.StartedSymbolAnalysisDto, error) {
	resolvedSymbol, err := symbolAnalysisService.symbolResolutionService.ResolveSymbol(ctx, dto.ResolveSymbolDto{Symbol: startSymbolAnalysisDto.Symbol, Category: startSymbolAnalysisDto.Category})
	if err != nil {
		return dto.StartedSymbolAnalysisDto{}, err
	}
	symbol := resolvedSymbol.Symbol
	// retried because a concurrent start may win the single-running-analysis index and then finish before we can reuse it
	for range maximumStartAttempts {
		now := symbolAnalysisService.clockProxy.Now()
		latestAnalysisEvent, findError := symbolAnalysisService.analysisEventRepository.FindLatestReusable(ctx, symbol.Value, symbol.Category.Value)
		if findError != nil {
			return dto.StartedSymbolAnalysisDto{}, fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, findError)
		}
		if latestAnalysisEvent != nil {
			latestAnalysisEventDomain := domains.NewAnalysisEventDomain(*latestAnalysisEvent)
			if latestAnalysisEventDomain.IsReusableAt(now) {
				reusedAnalysisEvent, dtoError := symbolAnalysisService.toDtoWithResult(ctx, *latestAnalysisEvent)
				return dto.StartedSymbolAnalysisDto{AnalysisEvent: reusedAnalysisEvent, IsNew: false, SearchKeyword: resolvedSymbol.SearchKeyword}, dtoError
			}
			if latestAnalysisEventDomain.IsStaleAt(now) {
				latestAnalysisEventDomain.Fail(FailureReasonTimedOut, now, vo.AnalystUsageVo{InputTokens: latestAnalysisEvent.InputTokens, OutputTokens: latestAnalysisEvent.OutputTokens})
				staleAnalysisEvent := latestAnalysisEventDomain.ToEntity()
				if updateError := symbolAnalysisService.analysisEventRepository.Update(ctx, &staleAnalysisEvent); updateError != nil {
					return dto.StartedSymbolAnalysisDto{}, fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, updateError)
				}
			}
		}
		if !startSymbolAnalysisDto.AllowsNewAnalysis {
			return dto.StartedSymbolAnalysisDto{}, ErrAnalysisCapacityReached
		}
		analysisEvent := domains.NewStartedAnalysisEventDomain(startSymbolAnalysisDto.ApiKeyID, symbol, symbolAnalysisService.analystConsultationService.ModelName(), now).ToEntity()
		createError := symbolAnalysisService.analysisEventRepository.Create(ctx, &analysisEvent)
		if errors.Is(createError, ErrAnalysisAlreadyRunning) {
			continue
		}
		if createError != nil {
			return dto.StartedSymbolAnalysisDto{}, fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, createError)
		}
		return dto.StartedSymbolAnalysisDto{AnalysisEvent: domains.NewAnalysisEventDomain(analysisEvent).ToDto(nil), IsNew: true, SearchKeyword: resolvedSymbol.SearchKeyword}, nil
	}
	return dto.StartedSymbolAnalysisDto{}, ErrAnalysisStorageUnavailable
}

func (symbolAnalysisService *SymbolAnalysisService) AnalyzeSymbol(ctx context.Context, analyzeSymbolDto dto.AnalyzeSymbolDto) error {
	storedAnalysisEvent, err := symbolAnalysisService.analysisEventRepository.FindByID(ctx, analyzeSymbolDto.AnalysisEventID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, err)
	}
	if storedAnalysisEvent == nil {
		return ErrAnalysisEventNotFound
	}
	analysisEvent := domains.NewAnalysisEventDomain(*storedAnalysisEvent)
	analysisContext, cancelAnalysis := context.WithTimeout(ctx, domains.AnalysisTimeout)
	defer cancelAnalysis()
	analystConsultation := symbolAnalysisService.analystConsultationService.Consult(analysisContext, dto.ConsultAnalystDto{Symbol: storedAnalysisEvent.Symbol, Category: storedAnalysisEvent.Category, SearchKeyword: analyzeSymbolDto.SearchKeyword})
	analysisEvent.RecordAnsweringModel(analystConsultation.AnsweringModel())
	failureReason := analystConsultation.FailureReason()
	if conclusion, isConcluded := analystConsultation.Conclusion(); isConcluded {
		priceQuote := symbolAnalysisService.priceSnapshotService.CapturePrice(analysisContext, dto.CapturePriceDto{Symbol: storedAnalysisEvent.Symbol, Category: storedAnalysisEvent.Category})
		analysisResult := conclusion.ToResultEntity(*storedAnalysisEvent, symbolAnalysisService.clockProxy.Now(), priceQuote)
		failureReason = ""
		if saveError := symbolAnalysisService.analysisResultRepository.Create(context.WithoutCancel(ctx), &analysisResult); saveError != nil {
			failureReason = FailureReasonResultNotSaved
		}
	}
	if failureReason == "" {
		analysisEvent.Succeed(symbolAnalysisService.clockProxy.Now(), analystConsultation.Usage())
	} else {
		analysisEvent.Fail(failureReason, symbolAnalysisService.clockProxy.Now(), analystConsultation.Usage())
	}
	finishedAnalysisEvent := analysisEvent.ToEntity()
	// recorded even when the analysis ran out of time
	if err := symbolAnalysisService.analysisEventRepository.Update(context.WithoutCancel(ctx), &finishedAnalysisEvent); err != nil {
		return fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, err)
	}
	return nil
}

func (symbolAnalysisService *SymbolAnalysisService) GetAnalysisEvent(ctx context.Context, analysisEventID uint) (dto.AnalysisEventDto, error) {
	storedAnalysisEvent, err := symbolAnalysisService.analysisEventRepository.FindByID(ctx, analysisEventID)
	if err != nil {
		return dto.AnalysisEventDto{}, fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, err)
	}
	if storedAnalysisEvent == nil {
		return dto.AnalysisEventDto{}, ErrAnalysisEventNotFound
	}
	return symbolAnalysisService.toDtoWithResult(ctx, *storedAnalysisEvent)
}

func (symbolAnalysisService *SymbolAnalysisService) FailInterruptedAnalysisEvents(ctx context.Context) error {
	if err := symbolAnalysisService.analysisEventRepository.FailAllRunning(ctx, FailureReasonInterruptedByRestart, symbolAnalysisService.clockProxy.Now()); err != nil {
		return fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, err)
	}
	return nil
}

func (symbolAnalysisService *SymbolAnalysisService) toDtoWithResult(ctx context.Context, analysisEvent entities.AnalysisEvent) (dto.AnalysisEventDto, error) {
	analysisEventDomain := domains.NewAnalysisEventDomain(analysisEvent)
	if !analysisEventDomain.IsSucceeded() {
		return analysisEventDomain.ToDto(nil), nil
	}
	analysisResult, err := symbolAnalysisService.analysisResultRepository.FindByAnalysisEventID(ctx, analysisEvent.ID)
	if err != nil {
		return dto.AnalysisEventDto{}, fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, err)
	}
	return analysisEventDomain.ToDto(analysisResult), nil
}
