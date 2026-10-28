package dto

import interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"

type SymbolDirectoryCatalogDto struct {
	// asked in order; the first directory that knows the stock wins
	TwStock []interfaces.IListedCompanyProxy
	// only names the company of a new analysis; a US symbol it does not know is still analysed
	UsStock interfaces.IListedCompanyProxy
	Crypto  interfaces.ICryptocurrencyProxy
}
