package vo

type ResolvedSymbolVo struct {
	Symbol         SymbolVo
	SearchKeyword  string
	RelevanceTerms []string
}

func NewTwStockResolvedSymbolVo(symbol SymbolVo, companyShortName string) ResolvedSymbolVo {
	return ResolvedSymbolVo{Symbol: symbol, SearchKeyword: companyShortName, RelevanceTerms: []string{companyShortName, symbol.Value}}
}

func NewCryptoResolvedSymbolVo(symbol SymbolVo, coinName string) ResolvedSymbolVo {
	return ResolvedSymbolVo{Symbol: symbol, SearchKeyword: coinName, RelevanceTerms: []string{coinName, symbol.Value}}
}

func NewUsStockResolvedSymbolVo(symbol SymbolVo) ResolvedSymbolVo {
	return ResolvedSymbolVo{Symbol: symbol, SearchKeyword: symbol.Value, RelevanceTerms: []string{symbol.Value}}
}
