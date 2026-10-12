package vo_test

import (
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestSymbolFilterVo(t *testing.T) {
	testCases := []struct {
		name                 string
		rawSymbol            string
		rawCategory          string
		expectedSymbolFilter vo.SymbolFilterVo
		expectedError        error
	}{
		{name: "neither given covers every symbol"},
		{name: "written the way analyses store it", rawSymbol: " btc ", rawCategory: "crypto", expectedSymbolFilter: vo.SymbolFilterVo{Symbol: "BTC", Category: "crypto"}},
		{name: "symbol without category", rawSymbol: "2330", expectedError: vo.ErrSymbolAndCategoryRequiredTogether},
		{name: "category without symbol", rawCategory: "twStock", expectedError: vo.ErrSymbolAndCategoryRequiredTogether},
		{name: "unsupported category", rawSymbol: "2330", rawCategory: "futures", expectedError: vo.ErrMarketCategoryUnsupported},
		{name: "blank symbol", rawSymbol: " ", rawCategory: "twStock", expectedError: vo.ErrSymbolRequired},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			symbolFilter, err := vo.NewSymbolFilterVo(testCase.rawSymbol, testCase.rawCategory)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedSymbolFilter, symbolFilter)
		})
	}
}
