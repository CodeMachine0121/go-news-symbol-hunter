package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const MaximumAnalystRounds = 5

type AnalystConsultationService struct {
	analystProxy      interfaces.IAnalystProxy
	newsSearchService *NewsSearchService
}

func NewAnalystConsultationService(analystProxy interfaces.IAnalystProxy, newsSearchService *NewsSearchService) *AnalystConsultationService {
	return &AnalystConsultationService{analystProxy: analystProxy, newsSearchService: newsSearchService}
}

func (analystConsultationService *AnalystConsultationService) ModelName() string {
	return analystConsultationService.analystProxy.ModelName()
}

func (analystConsultationService *AnalystConsultationService) Consult(ctx context.Context, consultAnalystDto dto.ConsultAnalystDto) domains.AnalystConsultationDomain {
	analystRequest := vo.AnalystRequestVo{Symbol: consultAnalystDto.Symbol, Category: consultAnalystDto.Category, SearchKeyword: consultAnalystDto.SearchKeyword, NewsPublishedWindow: consultAnalystDto.NewsPublishedWindow}
	analysisEvidence := domains.NewAnalysisEvidenceDomain()
	exchanges := []vo.AnalystExchangeVo{}
	usage := vo.AnalystUsageVo{}
	answeringModel := ""
	for round := range MaximumAnalystRounds {
		analystTurn, respondError := analystConsultationService.analystProxy.Respond(ctx, analystRequest, exchanges)
		usage = vo.AnalystUsageVo{InputTokens: usage.InputTokens + analystTurn.Usage.InputTokens, OutputTokens: usage.OutputTokens + analystTurn.Usage.OutputTokens}
		if errors.Is(respondError, context.DeadlineExceeded) {
			return domains.NewFailedAnalystConsultationDomain(FailureReasonTimedOut, usage, answeringModel)
		}
		if respondError != nil {
			return domains.NewFailedAnalystConsultationDomain(FailureReasonAnalystUnavailable, usage, answeringModel)
		}
		if analystTurn.ModelName != "" {
			answeringModel = analystTurn.ModelName
		}
		if analystTurn.IsRefused {
			return domains.NewFailedAnalystConsultationDomain(FailureReasonAnalystRefused, usage, answeringModel)
		}
		if analystTurn.Conclusion == nil && len(analystTurn.NewsSearches) == 0 {
			return domains.NewFailedAnalystConsultationDomain(FailureReasonAnalystIncomplete, usage, answeringModel)
		}
		if analystTurn.Conclusion != nil {
			conclusion, conclusionError := domains.NewAnalysisConclusionDomain(*analystTurn.Conclusion, analysisEvidence)
			if conclusionError != nil {
				return domains.NewFailedAnalystConsultationDomain(FailureReasonAnalystIncomplete, usage, answeringModel)
			}
			return domains.NewConcludedAnalystConsultationDomain(conclusion, usage, answeringModel)
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
				searchedNews[index], searchErrors[index] = analystConsultationService.newsSearchService.SearchSymbolNews(ctx, dto.SearchSymbolNewsDto{Symbol: newsSearch.Symbol, Category: newsSearch.Category, NewsPublishedWindow: consultAnalystDto.NewsPublishedWindow})
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
	return domains.NewFailedAnalystConsultationDomain(FailureReasonAnalystExceededRounds, usage, answeringModel)
}
