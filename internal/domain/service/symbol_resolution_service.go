package service

import (
	"context"
	"errors"
	"fmt"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type SymbolResolutionService struct {
	listedCompanyProxies []interfaces.IListedCompanyProxy
	cryptocurrencyProxy  interfaces.ICryptocurrencyProxy
}

func NewSymbolResolutionService(symbolDirectoryCatalog dto.SymbolDirectoryCatalogDto) *SymbolResolutionService {
	return &SymbolResolutionService{listedCompanyProxies: symbolDirectoryCatalog.TwStock, cryptocurrencyProxy: symbolDirectoryCatalog.Crypto}
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
		directoryErrors := []error{}
		for _, listedCompanyProxy := range symbolResolutionService.listedCompanyProxies {
			companyShortName, found, err := listedCompanyProxy.FindCompanyShortName(ctx, symbol.Value)
			if found {
				return vo.NewTwStockResolvedSymbolVo(symbol, companyShortName), nil
			}
			if err != nil {
				directoryErrors = append(directoryErrors, err)
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
		return vo.NewUsStockResolvedSymbolVo(symbol), nil
	}
}
