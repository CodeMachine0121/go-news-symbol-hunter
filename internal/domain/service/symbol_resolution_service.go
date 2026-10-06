package service

import (
	"fmt"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type SymbolResolutionService struct {
	listedCompanyProxy  interfaces.IListedCompanyProxy
	cryptocurrencyProxy interfaces.ICryptocurrencyProxy
}

func NewSymbolResolutionService(listedCompanyProxy interfaces.IListedCompanyProxy, cryptocurrencyProxy interfaces.ICryptocurrencyProxy) *SymbolResolutionService {
	return &SymbolResolutionService{listedCompanyProxy: listedCompanyProxy, cryptocurrencyProxy: cryptocurrencyProxy}
}

func (symbolResolutionService *SymbolResolutionService) ResolveSymbol(symbol vo.SymbolVo) (vo.ResolvedSymbolVo, error) {
	switch symbol.Category.Value {
	case vo.MarketCategoryTwStock:
		companyShortName, found, err := symbolResolutionService.listedCompanyProxy.FindCompanyShortName(symbol.Value)
		if err != nil {
			return vo.ResolvedSymbolVo{}, fmt.Errorf("%w: %v", ErrNewsProvidersUnavailable, err)
		}
		if !found {
			return vo.ResolvedSymbolVo{}, ErrSymbolNotFound
		}
		return vo.NewTwStockResolvedSymbolVo(symbol, companyShortName), nil
	case vo.MarketCategoryCrypto:
		coinName, found, err := symbolResolutionService.cryptocurrencyProxy.FindCoinName(symbol.Value)
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
