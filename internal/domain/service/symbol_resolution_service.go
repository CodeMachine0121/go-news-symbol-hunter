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
const usStockDirectoryLookupTimeout = 5 * time.Second

type SymbolResolutionService struct {
	twStockDirectoryProxies []interfaces.ISymbolDirectoryProxy
	usStockDirectoryProxy   interfaces.ISymbolDirectoryProxy
	cryptoDirectoryProxy    interfaces.ISymbolDirectoryProxy
}

func NewSymbolResolutionService(symbolDirectoryCatalog dto.SymbolDirectoryCatalogDto) *SymbolResolutionService {
	return &SymbolResolutionService{twStockDirectoryProxies: symbolDirectoryCatalog.TwStock, usStockDirectoryProxy: symbolDirectoryCatalog.UsStock, cryptoDirectoryProxy: symbolDirectoryCatalog.Crypto}
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
	switch category.Value {
	case vo.MarketCategoryTwStock:
		listings, directoryError := symbolResolutionService.searchTwStockListings(ctx, symbol.Value)
		resolvedSymbol, err := domains.NewSymbolDirectoryDomain(category, listings).FindBySymbol(symbol)
		// a directory we could not read might have listed the stock, so absence is not proven
		if err != nil && directoryError != nil {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, directoryError)
		}
		return resolvedSymbol, err
	case vo.MarketCategoryCrypto:
		listings, err := symbolResolutionService.cryptoDirectoryProxy.SearchSymbolListings(ctx, symbol.Value)
		if err != nil {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, err)
		}
		return domains.NewSymbolDirectoryDomain(category, listings).FindBySymbol(symbol)
	default:
		return vo.NewUsStockResolvedSymbolVo(symbol, ""), nil
	}
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
	switch category.Value {
	case vo.MarketCategoryTwStock:
		listings, directoryError := symbolResolutionService.searchTwStockListings(ctx, symbol.Value)
		resolvedSymbol, err := domains.NewSymbolDirectoryDomain(category, listings).Identify(symbol)
		// a name match is only unique when every directory was read, a symbol match is not affected
		if directoryError != nil && (err != nil || resolvedSymbol.Symbol.Value != symbol.Value) {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, directoryError)
		}
		return resolvedSymbol, err
	case vo.MarketCategoryCrypto:
		listings, err := symbolResolutionService.cryptoDirectoryProxy.SearchSymbolListings(ctx, symbol.Value)
		if err != nil {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, err)
		}
		return domains.NewSymbolDirectoryDomain(category, listings).Identify(symbol)
	default:
		lookupContext, cancelLookup := context.WithTimeout(ctx, usStockDirectoryLookupTimeout)
		defer cancelLookup()
		listings, err := symbolResolutionService.usStockDirectoryProxy.SearchSymbolListings(lookupContext, symbol.Value)
		// a US symbol was never rejected for an unreachable source, so the typed text is still analysed as a symbol
		if err != nil {
			return vo.NewUsStockResolvedSymbolVo(symbol, ""), nil
		}
		return domains.NewSymbolDirectoryDomain(category, listings).Identify(symbol)
	}
}

// asked together so a cold cache costs one download time
func (symbolResolutionService *SymbolResolutionService) searchTwStockListings(ctx context.Context, query string) ([]vo.SymbolListingVo, error) {
	listingsByDirectory := make([][]vo.SymbolListingVo, len(symbolResolutionService.twStockDirectoryProxies))
	lookupErrors := make([]error, len(symbolResolutionService.twStockDirectoryProxies))
	var waitGroup sync.WaitGroup
	for index, directoryProxy := range symbolResolutionService.twStockDirectoryProxies {
		waitGroup.Go(func() {
			listingsByDirectory[index], lookupErrors[index] = directoryProxy.SearchSymbolListings(ctx, query)
		})
	}
	waitGroup.Wait()
	listings := []vo.SymbolListingVo{}
	for _, directoryListings := range listingsByDirectory {
		listings = append(listings, directoryListings...)
	}
	return listings, errors.Join(lookupErrors...)
}
