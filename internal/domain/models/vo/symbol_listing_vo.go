package vo

// a lower priority is listed first: market cap rank for coins, result order for US stocks, none for Taiwan stocks
type SymbolListingVo struct {
	Symbol   string
	Name     string
	Priority int
}
