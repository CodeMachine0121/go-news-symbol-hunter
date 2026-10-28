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

// the company name is only for display; US news is still searched by ticker
func NewUsStockResolvedSymbolVo(symbol SymbolVo, companyName string) ResolvedSymbolVo {
	return ResolvedSymbolVo{Symbol: symbol, SymbolName: NewSymbolNameVo(companyName), SearchKeyword: symbol.Value, RelevanceTerms: []string{symbol.Value}}
}
