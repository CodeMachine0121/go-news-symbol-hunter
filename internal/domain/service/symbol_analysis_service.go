package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const MaximumAnalystRounds = 5

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
	category, err := vo.NewMarketCategoryVo(startSymbolAnalysisDto.Category)
	if err != nil {
		return dto.StartedSymbolAnalysisDto{}, err
	}
	symbol, err := vo.NewSymbolVo(startSymbolAnalysisDto.Symbol, category)
	if err != nil {
		return dto.StartedSymbolAnalysisDto{}, err
	}
	resolvedSymbol, err := symbolAnalysisService.symbolResolutionService.ResolveSymbol(ctx, symbol)
	if err != nil {
		return dto.StartedSymbolAnalysisDto{}, err
	}
	for attempt := range 2 {
		latestAnalysisEvent, findError := symbolAnalysisService.analysisEventRepository.FindLatestReusable(ctx, symbol.Value, category.Value)
		if findError != nil {
			return dto.StartedSymbolAnalysisDto{}, fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, findError)
		}
		if latestAnalysisEvent != nil && domains.NewAnalysisEventDomain(*latestAnalysisEvent).IsReusableAt(symbolAnalysisService.clockProxy.Now()) {
			reusedAnalysisEvent, dtoError := symbolAnalysisService.toDtoWithResult(ctx, *latestAnalysisEvent)
			return dto.StartedSymbolAnalysisDto{AnalysisEvent: reusedAnalysisEvent, IsNew: false, SearchKeyword: resolvedSymbol.SearchKeyword}, dtoError
		}
		if attempt > 0 {
			break
		}
		analysisEvent := domains.NewStartedAnalysisEventDomain(startSymbolAnalysisDto.ApiKeyID, symbol, symbolAnalysisService.analystProxy.ModelName(), symbolAnalysisService.clockProxy.Now()).ToEntity()
		createError := symbolAnalysisService.analysisEventRepository.Create(ctx, &analysisEvent)
		if errors.Is(createError, ErrAnalysisAlreadyRunning) {
			continue
		}
		if createError != nil {
			return dto.StartedSymbolAnalysisDto{}, fmt.Errorf("%w: %v", ErrAnalysisStorageUnavailable, createError)
		}
		return dto.StartedSymbolAnalysisDto{AnalysisEvent: domains.NewAnalysisEventDomain(analysisEvent).ToDto(nil), IsNew: true, SearchKeyword: resolvedSymbol.SearchKeyword}, nil
	}
	// a concurrent request created the running analysis and it finished as failed before we could reuse it
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
	analystRequest := vo.AnalystRequestVo{Symbol: storedAnalysisEvent.Symbol, Category: storedAnalysisEvent.Category, SearchKeyword: analyzeSymbolDto.SearchKeyword}
	analysisEvidence := domains.NewAnalysisEvidenceDomain()
	exchanges := []vo.AnalystExchangeVo{}
	usage := vo.AnalystUsageVo{}
	failureReason := FailureReasonAnalystExceededRounds
	isSucceeded := false
	for range MaximumAnalystRounds {
		analystTurn, respondError := symbolAnalysisService.analystProxy.Respond(ctx, analystRequest, exchanges)
		usage = vo.AnalystUsageVo{InputTokens: usage.InputTokens + analystTurn.Usage.InputTokens, OutputTokens: usage.OutputTokens + analystTurn.Usage.OutputTokens}
		if respondError != nil {
			failureReason = FailureReasonAnalystUnavailable
			break
		}
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
			if saveError := symbolAnalysisService.analysisResultRepository.Create(ctx, &analysisResult); saveError != nil {
				failureReason = FailureReasonResultNotSaved
				break
			}
			isSucceeded = true
			break
		}
		toolResults := make([]vo.AnalystToolResultVo, 0, len(analystTurn.NewsSearches))
		for _, newsSearch := range analystTurn.NewsSearches {
			symbolNews, searchError := symbolAnalysisService.newsSearchService.SearchSymbolNews(ctx, dto.SearchSymbolNewsDto{Symbol: newsSearch.Symbol, Category: newsSearch.Category})
			if searchError != nil {
				toolResults = append(toolResults, vo.AnalystToolResultVo{ToolCallID: newsSearch.ToolCallID, Content: searchError.Error(), IsError: true})
				continue
			}
			analysisEvidence.Record(symbolNews.News)
			searchResult, _ := json.Marshal(symbolNews)
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
	if err := symbolAnalysisService.analysisEventRepository.Update(ctx, &finishedAnalysisEvent); err != nil {
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
