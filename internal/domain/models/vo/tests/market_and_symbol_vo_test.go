package vo_test

import (
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMarketCategoryVo(t *testing.T) {
	testCases := []struct {
		name          string
		rawCategory   string
		expectedValue string
		expectedError error
	}{
		{name: "crypto", rawCategory: "crypto", expectedValue: "crypto"},
		{name: "taiwan stock", rawCategory: "twStock", expectedValue: "twStock"},
		{name: "us stock", rawCategory: "usStock", expectedValue: "usStock"},
		{name: "unsupported market", rawCategory: "港股", expectedError: vo.ErrMarketCategoryUnsupported},
		{name: "missing market", rawCategory: "", expectedError: vo.ErrMarketCategoryUnsupported},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			category, err := vo.NewMarketCategoryVo(testCase.rawCategory)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedValue, category.Value)
		})
	}
}

func TestNewSymbolVo(t *testing.T) {
	testCases := []struct {
		name          string
		rawSymbol     string
		category      string
		expectedValue string
		expectedError error
	}{
		{name: "us stock is trimmed and upper-cased", rawSymbol: " aapl ", category: "usStock", expectedValue: "AAPL"},
		{name: "crypto is upper-cased", rawSymbol: "btc", category: "crypto", expectedValue: "BTC"},
		{name: "taiwan stock code is trimmed", rawSymbol: " 2330 ", category: "twStock", expectedValue: "2330"},
		{name: "missing symbol", rawSymbol: "", category: "crypto", expectedError: vo.ErrSymbolRequired},
		{name: "blank symbol", rawSymbol: "   ", category: "usStock", expectedError: vo.ErrSymbolRequired},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			category, err := vo.NewMarketCategoryVo(testCase.category)
			require.NoError(t, err)

			symbol, err := vo.NewSymbolVo(testCase.rawSymbol, category)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedValue, symbol.Value)
		})
	}
}

func TestResolvedSymbolVo_SearchKeywordsAndRelevanceTerms(t *testing.T) {
	twStock, _ := vo.NewMarketCategoryVo("twStock")
	crypto, _ := vo.NewMarketCategoryVo("crypto")
	usStock, _ := vo.NewMarketCategoryVo("usStock")
	twStockSymbol, _ := vo.NewSymbolVo("2330", twStock)
	cryptoSymbol, _ := vo.NewSymbolVo("BTC", crypto)
	usStockSymbol, _ := vo.NewSymbolVo("AAPL", usStock)

	assert.Equal(t, vo.ResolvedSymbolVo{Symbol: twStockSymbol, SearchKeyword: "台積電", RelevanceTerms: []string{"台積電", "2330"}}, vo.NewTwStockResolvedSymbolVo(twStockSymbol, "台積電"))
	assert.Equal(t, vo.ResolvedSymbolVo{Symbol: cryptoSymbol, SearchKeyword: "Bitcoin", RelevanceTerms: []string{"Bitcoin", "BTC"}}, vo.NewCryptoResolvedSymbolVo(cryptoSymbol, "Bitcoin"))
	assert.Equal(t, vo.ResolvedSymbolVo{Symbol: usStockSymbol, SearchKeyword: "AAPL", RelevanceTerms: []string{"AAPL"}}, vo.NewUsStockResolvedSymbolVo(usStockSymbol))
}
