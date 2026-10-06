package vo

import (
	"errors"
	"strings"
)

var ErrSymbolRequired = errors.New("標的為必填")

type SymbolVo struct {
	Value    string
	Category MarketCategoryVo
}

func NewSymbolVo(rawSymbol string, category MarketCategoryVo) (SymbolVo, error) {
	trimmedSymbol := strings.TrimSpace(rawSymbol)
	if trimmedSymbol == "" {
		return SymbolVo{}, ErrSymbolRequired
	}
	if category.Value != MarketCategoryTwStock {
		trimmedSymbol = strings.ToUpper(trimmedSymbol)
	}
	return SymbolVo{Value: trimmedSymbol, Category: category}, nil
}
