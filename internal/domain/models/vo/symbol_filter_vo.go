package vo

import "errors"

var ErrSymbolAndCategoryRequiredTogether = errors.New("標的與市場類別需一起提供")

// narrows a query to one symbol in one market; blank Symbol and Category mean every symbol
type SymbolFilterVo struct {
	Symbol   string
	Category string
}

// the symbol is written the way analyses store it, so " btc " finds BTC
func NewSymbolFilterVo(rawSymbol string, rawCategory string) (SymbolFilterVo, error) {
	if rawSymbol == "" && rawCategory == "" {
		return SymbolFilterVo{}, nil
	}
	if rawSymbol == "" || rawCategory == "" {
		return SymbolFilterVo{}, ErrSymbolAndCategoryRequiredTogether
	}
	marketCategory, err := NewMarketCategoryVo(rawCategory)
	if err != nil {
		return SymbolFilterVo{}, err
	}
	symbol, err := NewSymbolVo(rawSymbol, marketCategory)
	if err != nil {
		return SymbolFilterVo{}, err
	}
	return SymbolFilterVo{Symbol: symbol.Value, Category: marketCategory.Value}, nil
}
