package service

import (
	"context"
	"fmt"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type SymbolResolutionService struct {
	listedCompanyProxy  interfaces.IListedCompanyProxy
	cryptocurrencyProxy interfaces.ICryptocurrencyProxy
}

func NewSymbolResolutionService(listedCompanyProxy interfaces.IListedCompanyProxy, cryptocurrencyProxy interfaces.ICryptocurrencyProxy) *SymbolResolutionService {
	return &SymbolResolutionService{listedCompanyProxy: listedCompanyProxy, cryptocurrencyProxy: cryptocurrencyProxy}
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
		companyShortName, found, err := symbolResolutionService.listedCompanyProxy.FindCompanyShortName(ctx, symbol.Value)
		if err != nil {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, err)
		}
		if !found {
			return vo.ResolvedSymbolVo{}, ErrSymbolNotFound
		}
		return vo.NewTwStockResolvedSymbolVo(symbol, companyShortName), nil
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
