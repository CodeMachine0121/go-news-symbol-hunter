package dto

import interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"

// a nil source means the market cannot be backtested
type PriceHistoryCatalogDto struct {
	TwStock interfaces.IPriceHistoryProxy
	UsStock interfaces.IPriceHistoryProxy
	Crypto  interfaces.IPriceHistoryProxy
}
