package vo

type ResolvedSymbolVo struct {
	Symbol         SymbolVo
	SymbolName     SymbolNameVo
	SearchKeyword  string
	RelevanceTerms []string
}

func NewTwStockResolvedSymbolVo(symbol SymbolVo, companyShortName string) ResolvedSymbolVo {
	return ResolvedSymbolVo{Symbol: symbol, SymbolName: NewSymbolNameVo(companyShortName), SearchKeyword: companyShortName, RelevanceTerms: []string{companyShortName, symbol.Value}}
}

func NewCryptoResolvedSymbolVo(symbol SymbolVo, coinName string) ResolvedSymbolVo {
	return ResolvedSymbolVo{Symbol: symbol, SymbolName: NewSymbolNameVo(coinName), SearchKeyword: coinName, RelevanceTerms: []string{coinName, symbol.Value}}
}

// a US company name is looked up only when an analysis is created, so it starts empty
func NewUsStockResolvedSymbolVo(symbol SymbolVo) ResolvedSymbolVo {
	return ResolvedSymbolVo{Symbol: symbol, SearchKeyword: symbol.Value, RelevanceTerms: []string{symbol.Value}}
}
