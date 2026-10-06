package service

import (
	"sync"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type NewsSearchService struct {
	symbolResolutionService *SymbolResolutionService
	clockProxy              interfaces.IClockProxy
	newsProvidersByCategory map[string][]dto.NewsProviderDto
}

type newsProviderOutcome struct {
	providerName   string
	newsCollection domains.NewsCollectionDomain
	err            error
}

func NewNewsSearchService(symbolResolutionService *SymbolResolutionService, clockProxy interfaces.IClockProxy, newsProvidersByCategory map[string][]dto.NewsProviderDto) *NewsSearchService {
	return &NewsSearchService{
		symbolResolutionService: symbolResolutionService,
		clockProxy:              clockProxy,
		newsProvidersByCategory: newsProvidersByCategory,
	}
}

func (newsSearchService *NewsSearchService) SearchSymbolNews(searchSymbolNewsDto dto.SearchSymbolNewsDto) (dto.SymbolNewsDto, error) {
	category, err := vo.NewMarketCategoryVo(searchSymbolNewsDto.Category)
	if err != nil {
		return dto.SymbolNewsDto{}, err
	}
	symbol, err := vo.NewSymbolVo(searchSymbolNewsDto.Symbol, category)
	if err != nil {
		return dto.SymbolNewsDto{}, err
	}
	resolvedSymbol, err := newsSearchService.symbolResolutionService.ResolveSymbol(symbol)
	if err != nil {
		return dto.SymbolNewsDto{}, err
	}

	newsProviders := newsSearchService.newsProvidersByCategory[category.Value]
	outcomes := make([]newsProviderOutcome, len(newsProviders))
	var waitGroup sync.WaitGroup
	for index, newsProvider := range newsProviders {
		waitGroup.Go(func() {
			news, fetchError := newsProvider.NewsProxy.FetchNews(resolvedSymbol.SearchKeyword)
			newsCollection := domains.NewNewsCollectionDomain(news)
			if newsProvider.RequiresRelevanceFilter {
				newsCollection = newsCollection.KeepMentioning(resolvedSymbol.RelevanceTerms)
			}
			outcomes[index] = newsProviderOutcome{providerName: newsProvider.NewsProxy.ProviderName(), newsCollection: newsCollection, err: fetchError}
		})
	}
	waitGroup.Wait()

	mergedNewsCollection := domains.NewNewsCollectionDomain(nil)
	failedNewsProviders := []string{}
	for _, outcome := range outcomes {
		if outcome.err != nil {
			failedNewsProviders = append(failedNewsProviders, outcome.providerName)
			continue
		}
		mergedNewsCollection = mergedNewsCollection.Merge(outcome.newsCollection)
	}
	if len(failedNewsProviders) == len(newsProviders) {
		return dto.SymbolNewsDto{}, ErrNewsProvidersUnavailable
	}
	return dto.SymbolNewsDto{
		Symbol:              symbol.Value,
		Category:            category.Value,
		News:                mergedNewsCollection.Curate(newsSearchService.clockProxy.Now()).ToDtos(),
		FailedNewsProviders: failedNewsProviders,
	}, nil
}
