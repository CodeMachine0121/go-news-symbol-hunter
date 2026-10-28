package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

// a US company name is only for display, so waiting longer would only slow the start of an analysis
const usStockCompanyNameLookupTimeout = 5 * time.Second

type SymbolResolutionService struct {
	listedCompanyProxies      []interfaces.IListedCompanyProxy
	usStockListedCompanyProxy interfaces.IListedCompanyProxy
	cryptocurrencyProxy       interfaces.ICryptocurrencyProxy
}

func NewSymbolResolutionService(symbolDirectoryCatalog dto.SymbolDirectoryCatalogDto) *SymbolResolutionService {
	return &SymbolResolutionService{listedCompanyProxies: symbolDirectoryCatalog.TwStock, usStockListedCompanyProxy: symbolDirectoryCatalog.UsStock, cryptocurrencyProxy: symbolDirectoryCatalog.Crypto}
}

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
		// asked together so a cold cache costs one download time, then read in priority order
		companyShortNames := make([]string, len(symbolResolutionService.listedCompanyProxies))
		foundInDirectories := make([]bool, len(symbolResolutionService.listedCompanyProxies))
		lookupErrors := make([]error, len(symbolResolutionService.listedCompanyProxies))
		var waitGroup sync.WaitGroup
		for index, listedCompanyProxy := range symbolResolutionService.listedCompanyProxies {
			waitGroup.Go(func() {
				companyShortNames[index], foundInDirectories[index], lookupErrors[index] = listedCompanyProxy.FindCompanyShortName(ctx, symbol.Value)
			})
		}
		waitGroup.Wait()
		directoryErrors := []error{}
		for index := range symbolResolutionService.listedCompanyProxies {
			if foundInDirectories[index] {
				return vo.NewTwStockResolvedSymbolVo(symbol, companyShortNames[index]), nil
			}
			if lookupErrors[index] != nil {
				directoryErrors = append(directoryErrors, lookupErrors[index])
			}
		}
		// a directory we could not read might have listed the stock, so absence is not proven
		if len(directoryErrors) > 0 {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, errors.Join(directoryErrors...))
		}
		return vo.ResolvedSymbolVo{}, ErrSymbolNotFound
	case vo.MarketCategoryCrypto:
		coinName, found, err := symbolResolutionService.cryptocurrencyProxy.FindCoinName(ctx, symbol.Value)
		if err != nil {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, err)
		}
		if !found {
			return vo.ResolvedSymbolVo{}, ErrSymbolNotFound
		}
		return vo.NewCryptoResolvedSymbolVo(symbol, coinName), nil
	default:
		lookupContext, cancelLookup := context.WithTimeout(ctx, usStockCompanyNameLookupTimeout)
		defer cancelLookup()
		// an unknown or unreachable name leaves it empty; a US symbol is never rejected for it
		companyName, found, err := symbolResolutionService.usStockListedCompanyProxy.FindCompanyShortName(lookupContext, symbol.Value)
		if err != nil || !found {
			companyName = ""
		}
		return vo.NewUsStockResolvedSymbolVo(symbol, companyName), nil
	}
}
