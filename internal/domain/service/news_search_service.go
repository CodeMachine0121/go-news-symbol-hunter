package service

import (
	"context"
	"fmt"
	"sync"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type NewsSearchService struct {
	symbolResolutionService *SymbolResolutionService
	clockProxy              interfaces.IClockProxy
	newsProviderCatalog     dto.NewsProviderCatalogDto
}

type newsProviderOutcome struct {
	providerName   string
	newsCollection domains.NewsCollectionDomain
	err            error
}

func NewNewsSearchService(symbolResolutionService *SymbolResolutionService, clockProxy interfaces.IClockProxy, newsProviderCatalog dto.NewsProviderCatalogDto) *NewsSearchService {
	return &NewsSearchService{
		symbolResolutionService: symbolResolutionService,
		clockProxy:              clockProxy,
		newsProviderCatalog:     newsProviderCatalog,
	}
}

func (newsSearchService *NewsSearchService) SearchSymbolNews(ctx context.Context, searchSymbolNewsDto dto.SearchSymbolNewsDto) (dto.SymbolNewsDto, error) {
	category, err := vo.NewMarketCategoryVo(searchSymbolNewsDto.Category)
	if err != nil {
		return dto.SymbolNewsDto{}, err
	}
	symbol, err := vo.NewSymbolVo(searchSymbolNewsDto.Symbol, category)
	if err != nil {
		return dto.SymbolNewsDto{}, err
	}
	resolvedSymbol, err := newsSearchService.symbolResolutionService.ResolveSymbol(ctx, symbol)
	if err != nil {
		return dto.SymbolNewsDto{}, err
	}

	newsProviders := newsSearchService.newsProviderCatalog.UsStock
	switch category.Value {
	case vo.MarketCategoryTwStock:
		newsProviders = newsSearchService.newsProviderCatalog.TwStock
	case vo.MarketCategoryCrypto:
		newsProviders = newsSearchService.newsProviderCatalog.Crypto
	}
	outcomes := make([]newsProviderOutcome, len(newsProviders))
	var waitGroup sync.WaitGroup
	for index, newsProvider := range newsProviders {
		waitGroup.Go(func() {
			providerName := newsProvider.NewsProxy.ProviderName()
			// a panicking provider must fail alone instead of taking down the server
			defer func() {
				if recovered := recover(); recovered != nil {
					outcomes[index] = newsProviderOutcome{providerName: providerName, err: fmt.Errorf("news provider panicked: %v", recovered)}
				}
			}()
			news, fetchError := newsProvider.NewsProxy.FetchNews(ctx, resolvedSymbol.SearchKeyword)
			newsCollection := domains.NewNewsCollectionDomain(news)
			if newsProvider.RequiresRelevanceFilter {
				newsCollection = newsCollection.KeepMentioning(resolvedSymbol.RelevanceTerms)
			}
			outcomes[index] = newsProviderOutcome{providerName: providerName, newsCollection: newsCollection, err: fetchError}
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
