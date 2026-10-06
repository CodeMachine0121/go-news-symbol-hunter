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
	normalizedSymbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
	if normalizedSymbol == "" {
		return SymbolVo{}, ErrSymbolRequired
	}
	return SymbolVo{Value: normalizedSymbol, Category: category}, nil
}
