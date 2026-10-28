package dto

import interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"

type SymbolDirectoryCatalogDto struct {
	TwStock []interfaces.ISymbolDirectoryProxy
	UsStock interfaces.ISymbolDirectoryProxy
	Crypto  interfaces.ISymbolDirectoryProxy
}
