package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	MaximumAnalystRounds = 5
	maximumStartAttempts = 3
)

type SymbolAnalysisService struct {
	symbolResolutionService  *SymbolResolutionService
	newsSearchService        *NewsSearchService
	analystProxy             interfaces.IAnalystProxy
	analysisEventRepository  interfaces.IAnalysisEventRepository
	analysisResultRepository interfaces.IAnalysisResultRepository
	clockProxy               interfaces.IClockProxy
}

func NewSymbolAnalysisService(symbolResolutionService *SymbolResolutionService, newsSearchService *NewsSearchService, analystProxy interfaces.IAnalystProxy, analysisEventRepository interfaces.IAnalysisEventRepository, analysisResultRepository interfaces.IAnalysisResultRepository, clockProxy interfaces.IClockProxy) *SymbolAnalysisService {
	return &SymbolAnalysisService{
		symbolResolutionService:  symbolResolutionService,
		newsSearchService:        newsSearchService,
		analystProxy:             analystProxy,
		analysisEventRepository:  analysisEventRepository,
		analysisResultRepository: analysisResultRepository,
		clockProxy:               clockProxy,
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
		analysisEvent := domains.NewStartedAnalysisEventDomain(startSymbolAnalysisDto.ApiKeyID, symbol, symbolAnalysisService.analystProxy.ModelName(), now).ToEntity()
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
	analystRequest := vo.AnalystRequestVo{Symbol: storedAnalysisEvent.Symbol, Category: storedAnalysisEvent.Category, SearchKeyword: analyzeSymbolDto.SearchKeyword}
	analysisEvidence := domains.NewAnalysisEvidenceDomain()
	exchanges := []vo.AnalystExchangeVo{}
	usage := vo.AnalystUsageVo{}
	failureReason := FailureReasonAnalystExceededRounds
	isSucceeded := false
	for round := range MaximumAnalystRounds {
		analystTurn, respondError := symbolAnalysisService.analystProxy.Respond(analysisContext, analystRequest, exchanges)
		usage = vo.AnalystUsageVo{InputTokens: usage.InputTokens + analystTurn.Usage.InputTokens, OutputTokens: usage.OutputTokens + analystTurn.Usage.OutputTokens}
		if errors.Is(respondError, context.DeadlineExceeded) {
			failureReason = FailureReasonTimedOut
			break
		}
		if respondError != nil {
			failureReason = FailureReasonAnalystUnavailable
			break
		}
		analysisEvent.RecordAnsweringModel(analystTurn.ModelName)
		if analystTurn.IsRefused {
			failureReason = FailureReasonAnalystRefused
			break
		}
		if analystTurn.Conclusion == nil && len(analystTurn.NewsSearches) == 0 {
			failureReason = FailureReasonAnalystIncomplete
			break
		}
		if analystTurn.Conclusion != nil {
			conclusion, conclusionError := domains.NewAnalysisConclusionDomain(*analystTurn.Conclusion, analysisEvidence)
			if conclusionError != nil {
				failureReason = FailureReasonAnalystIncomplete
				break
			}
			analysisResult := conclusion.ToResultEntity(*storedAnalysisEvent, symbolAnalysisService.clockProxy.Now())
			if saveError := symbolAnalysisService.analysisResultRepository.Create(context.WithoutCancel(ctx), &analysisResult); saveError != nil {
				failureReason = FailureReasonResultNotSaved
				break
			}
			isSucceeded = true
			break
		}
		// the analyst could never read search results requested in the final round
		if round == MaximumAnalystRounds-1 {
			break
		}
		searchedNews := make([]dto.SymbolNewsDto, len(analystTurn.NewsSearches))
		searchErrors := make([]error, len(analystTurn.NewsSearches))
		var waitGroup sync.WaitGroup
		for index, newsSearch := range analystTurn.NewsSearches {
			waitGroup.Go(func() {
				searchedNews[index], searchErrors[index] = symbolAnalysisService.newsSearchService.SearchSymbolNews(analysisContext, dto.SearchSymbolNewsDto{Symbol: newsSearch.Symbol, Category: newsSearch.Category})
			})
		}
		waitGroup.Wait()
		toolResults := make([]vo.AnalystToolResultVo, 0, len(analystTurn.NewsSearches))
		for index, newsSearch := range analystTurn.NewsSearches {
			if searchErrors[index] != nil {
				toolResults = append(toolResults, vo.AnalystToolResultVo{ToolCallID: newsSearch.ToolCallID, Content: searchErrors[index].Error(), IsError: true})
				continue
			}
			analysisEvidence.Record(searchedNews[index].News)
			searchResult, _ := json.Marshal(searchedNews[index])
			toolResults = append(toolResults, vo.AnalystToolResultVo{ToolCallID: newsSearch.ToolCallID, Content: string(searchResult)})
		}
		exchanges = append(exchanges, vo.AnalystExchangeVo{Reply: analystTurn.Reply, ToolResults: toolResults})
	}
	if isSucceeded {
		analysisEvent.Succeed(symbolAnalysisService.clockProxy.Now(), usage)
	} else {
		analysisEvent.Fail(failureReason, symbolAnalysisService.clockProxy.Now(), usage)
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
