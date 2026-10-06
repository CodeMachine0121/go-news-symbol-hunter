package dto

import interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"

type PriceProviderCatalogDto struct {
	TwStock interfaces.IPriceProxy
	UsStock interfaces.IPriceProxy
	Crypto  interfaces.IPriceProxy
}
