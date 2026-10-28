package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

// an unreachable US directory only falls back to the typed symbol, so waiting longer would only slow the start of an analysis
var directoryLookupTimeouts = map[string]time.Duration{vo.MarketCategoryUsStock: 5 * time.Second}

type SymbolResolutionService struct {
	directoryProxiesByCategory map[string][]interfaces.ISymbolDirectoryProxy
}

func NewSymbolResolutionService(symbolDirectoryCatalog dto.SymbolDirectoryCatalogDto) *SymbolResolutionService {
	return &SymbolResolutionService{directoryProxiesByCategory: map[string][]interfaces.ISymbolDirectoryProxy{
		vo.MarketCategoryTwStock: symbolDirectoryCatalog.TwStock,
		vo.MarketCategoryUsStock: {symbolDirectoryCatalog.UsStock},
		vo.MarketCategoryCrypto:  {symbolDirectoryCatalog.Crypto},
	}}
}

// only a symbol is accepted; US symbols are not looked up so searching news never waits on the US directory
func (symbolResolutionService *SymbolResolutionService) ResolveSymbol(ctx context.Context, resolveSymbolDto dto.ResolveSymbolDto) (vo.ResolvedSymbolVo, error) {
	category, err := vo.NewMarketCategoryVo(resolveSymbolDto.Category)
	if err != nil {
		return vo.ResolvedSymbolVo{}, err
	}
	symbol, err := vo.NewSymbolVo(resolveSymbolDto.Symbol, category)
	if err != nil {
		return vo.ResolvedSymbolVo{}, err
	}
	if category.Value == vo.MarketCategoryUsStock {
		return vo.NewUsStockResolvedSymbolVo(symbol, ""), nil
	}
	symbolDirectory, directoryError := symbolResolutionService.searchDirectories(ctx, category, symbol.Value)
	if resolvedSymbol, isFound := symbolDirectory.FindBySymbol(symbol); isFound {
		return resolvedSymbol, nil
	}
	// a directory we could not read might have listed the symbol, so absence is not proven
	if directoryError != nil {
		return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, directoryError)
	}
	return vo.ResolvedSymbolVo{}, ErrSymbolNotFound
}

// the typed text may be a symbol or a name; the result always carries the symbol
func (symbolResolutionService *SymbolResolutionService) IdentifySymbol(ctx context.Context, resolveSymbolDto dto.ResolveSymbolDto) (vo.ResolvedSymbolVo, error) {
	category, err := vo.NewMarketCategoryVo(resolveSymbolDto.Category)
	if err != nil {
		return vo.ResolvedSymbolVo{}, err
	}
	symbol, err := vo.NewSymbolVo(resolveSymbolDto.Symbol, category)
	if err != nil {
		return vo.ResolvedSymbolVo{}, err
	}
	symbolDirectory, directoryError := symbolResolutionService.searchDirectories(ctx, category, symbol.Value)
	// a US symbol was never rejected for an unreachable source, so the typed text is still analysed as a symbol
	if directoryError != nil && category.Value == vo.MarketCategoryUsStock {
		return vo.NewUsStockResolvedSymbolVo(symbol, ""), nil
	}
	resolvedSymbol, candidates, isIdentified := symbolDirectory.Identify(symbol)
	// a name match is only unique when every directory was read, a symbol match is not affected
	if directoryError != nil && (!isIdentified || resolvedSymbol.Symbol.Value != symbol.Value) {
		return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, directoryError)
	}
	if isIdentified {
		return resolvedSymbol, nil
	}
	if len(candidates) > 0 {
		return vo.ResolvedSymbolVo{}, &SymbolAmbiguousError{Candidates: candidates}
	}
	return vo.ResolvedSymbolVo{}, ErrSymbolNotFound
}

// asked together so a cold cache costs one download time; what the readable directories list is kept even when another fails
func (symbolResolutionService *SymbolResolutionService) searchDirectories(ctx context.Context, category vo.MarketCategoryVo, query string) (domains.SymbolDirectoryDomain, error) {
	lookupContext, cancelLookup := context.WithCancel(ctx)
	defer cancelLookup()
	if lookupTimeout, isLimited := directoryLookupTimeouts[category.Value]; isLimited {
		limitedLookupContext, cancelLimitedLookup := context.WithTimeout(lookupContext, lookupTimeout)
		defer cancelLimitedLookup()
		lookupContext = limitedLookupContext
	}
	directoryProxies := symbolResolutionService.directoryProxiesByCategory[category.Value]
	listingsByDirectory := make([][]vo.SymbolListingVo, len(directoryProxies))
	lookupErrors := make([]error, len(directoryProxies))
	var waitGroup sync.WaitGroup
	for index, directoryProxy := range directoryProxies {
		waitGroup.Go(func() {
			listingsByDirectory[index], lookupErrors[index] = directoryProxy.SearchSymbolListings(lookupContext, query)
		})
	}
	waitGroup.Wait()
	listings := []vo.SymbolListingVo{}
	for _, directoryListings := range listingsByDirectory {
		listings = append(listings, directoryListings...)
	}
	return domains.NewSymbolDirectoryDomain(category, listings), errors.Join(lookupErrors...)
}
