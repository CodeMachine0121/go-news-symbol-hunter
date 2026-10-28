package vo

import "strings"

const maximumSymbolNameLength = 100

type SymbolNameVo struct {
	Value string
}

// names come from outside sources, so they are trimmed and cut to what storage holds
func NewSymbolNameVo(rawSymbolName string) SymbolNameVo {
	symbolName := []rune(strings.TrimSpace(rawSymbolName))
	if len(symbolName) > maximumSymbolNameLength {
		symbolName = symbolName[:maximumSymbolNameLength]
	}
	return SymbolNameVo{Value: strings.TrimSpace(string(symbolName))}
}
