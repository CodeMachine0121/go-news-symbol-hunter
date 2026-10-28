package domains_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	twStockCategory = vo.MarketCategoryVo{Value: vo.MarketCategoryTwStock}
	cryptoCategory  = vo.MarketCategoryVo{Value: vo.MarketCategoryCrypto}
	usStockCategory = vo.MarketCategoryVo{Value: vo.MarketCategoryUsStock}
)

var twStockListings = []vo.SymbolListingVo{
	{Symbol: "2330", Name: "台積電"},
	{Symbol: "0050", Name: "元大台灣50"},
	{Symbol: "00631L", Name: "元大台灣50正2"},
	{Symbol: "00679B", Name: "元大美債20年"},
}

var cryptoListings = []vo.SymbolListingVo{
	{Symbol: "bitcoin", Name: "HarryPotterObamaSonic10Inu", Priority: 500},
	{Symbol: "bch", Name: "Bitcoin Cash", Priority: 21},
	{Symbol: "btc", Name: "Bitcoin", Priority: 1},
	{Symbol: "unranked", Name: "Unranked Bitcoin", Priority: math.MaxInt},
}

var usStockListings = []vo.SymbolListingVo{
	{Symbol: "APLE", Name: "Apple Hospitality REIT, Inc.", Priority: 0},
	{Symbol: "AAPL", Name: "Apple Inc.", Priority: 1},
	{Symbol: "TSLA", Name: "Tesla, Inc.", Priority: 2},
	{Symbol: "SPY", Name: "State Street SPDR S&P 500 ETF Trust", Priority: 3},
	{Symbol: "SPYX", Name: "State Street SPDR S&P 500 Fossil Fuel Reserves Free ETF", Priority: 4},
	{Symbol: "EFIV", Name: "State Street SPDR S&P 500 ESG ETF", Priority: 5},
}

func identify(t *testing.T, category vo.MarketCategoryVo, listings []vo.SymbolListingVo, typedText string) (vo.ResolvedSymbolVo, error) {
	symbol, err := vo.NewSymbolVo(typedText, category)
	require.NoError(t, err)
	return domains.NewSymbolDirectoryDomain(category, listings).Identify(symbol)
}

func TestSymbolDirectoryDomain_IdentifiesTheSymbolBehindATypedSymbolOrName(t *testing.T) {
	testCases := []struct {
		name                  string
		category              vo.MarketCategoryVo
		listings              []vo.SymbolListingVo
		typedText             string
		expectedSymbol        string
		expectedSymbolName    string
		expectedSearchKeyword string
	}{
		{name: "Taiwan stock by symbol", category: twStockCategory, listings: twStockListings, typedText: "2330", expectedSymbol: "2330", expectedSymbolName: "台積電", expectedSearchKeyword: "台積電"},
		{name: "Taiwan stock by company short name", category: twStockCategory, listings: twStockListings, typedText: "台積電", expectedSymbol: "2330", expectedSymbolName: "台積電", expectedSearchKeyword: "台積電"},
		{name: "listed ETF by symbol", category: twStockCategory, listings: twStockListings, typedText: "0050", expectedSymbol: "0050", expectedSymbolName: "元大台灣50", expectedSearchKeyword: "元大台灣50"},
		{name: "OTC ETF by name", category: twStockCategory, listings: twStockListings, typedText: "元大美債20年", expectedSymbol: "00679B", expectedSymbolName: "元大美債20年", expectedSearchKeyword: "元大美債20年"},
		{name: "symbol with letters typed in lower case", category: twStockCategory, listings: twStockListings, typedText: "00679b", expectedSymbol: "00679B", expectedSymbolName: "元大美債20年", expectedSearchKeyword: "元大美債20年"},
		{name: "the only name containing the typed text", category: twStockCategory, listings: twStockListings, typedText: "台積", expectedSymbol: "2330", expectedSymbolName: "台積電", expectedSearchKeyword: "台積電"},
		{name: "an exact name wins over names containing it", category: twStockCategory, listings: twStockListings, typedText: "元大台灣50", expectedSymbol: "0050", expectedSymbolName: "元大台灣50", expectedSearchKeyword: "元大台灣50"},
		{name: "coin by symbol", category: cryptoCategory, listings: cryptoListings, typedText: "BTC", expectedSymbol: "BTC", expectedSymbolName: "Bitcoin", expectedSearchKeyword: "Bitcoin"},
		{name: "coin by name in any letter case", category: cryptoCategory, listings: cryptoListings, typedText: "bitcoin", expectedSymbol: "BTC", expectedSymbolName: "Bitcoin", expectedSearchKeyword: "Bitcoin"},
		{name: "the best market cap rank wins among exact symbols and names", category: cryptoCategory, listings: cryptoListings, typedText: "Bitcoin", expectedSymbol: "BTC", expectedSymbolName: "Bitcoin", expectedSearchKeyword: "Bitcoin"},
		{name: "US stock by symbol keeps searching by symbol", category: usStockCategory, listings: usStockListings, typedText: "TSLA", expectedSymbol: "TSLA", expectedSymbolName: "Tesla, Inc.", expectedSearchKeyword: "TSLA"},
		{name: "US stock by a name typed like a symbol", category: usStockCategory, listings: usStockListings, typedText: "TESLA", expectedSymbol: "TSLA", expectedSymbolName: "Tesla, Inc.", expectedSearchKeyword: "TSLA"},
		{name: "US company name equal once company suffixes are left out", category: usStockCategory, listings: usStockListings, typedText: "Apple", expectedSymbol: "AAPL", expectedSymbolName: "Apple Inc.", expectedSearchKeyword: "AAPL"},
		{name: "US company name typed with its suffix", category: usStockCategory, listings: usStockListings, typedText: "apple inc", expectedSymbol: "AAPL", expectedSymbolName: "Apple Inc.", expectedSearchKeyword: "AAPL"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolvedSymbol, err := identify(t, testCase.category, testCase.listings, testCase.typedText)

			require.NoError(t, err)
			assert.Equal(t, vo.SymbolVo{Value: testCase.expectedSymbol, Category: testCase.category}, resolvedSymbol.Symbol)
			assert.Equal(t, testCase.expectedSymbolName, resolvedSymbol.SymbolName.Value)
			assert.Equal(t, testCase.expectedSearchKeyword, resolvedSymbol.SearchKeyword)
		})
	}
}

func TestSymbolDirectoryDomain_ListsCandidatesWhenSeveralNamesContainTheTypedText(t *testing.T) {
	testCases := []struct {
		name               string
		category           vo.MarketCategoryVo
		listings           []vo.SymbolListingVo
		typedText          string
		expectedCandidates []dto.SymbolCandidateDto
	}{
		{name: "coins in market cap order", category: cryptoCategory, listings: cryptoListings, typedText: "Bitco", expectedCandidates: []dto.SymbolCandidateDto{
			{Symbol: "BTC", SymbolName: "Bitcoin"}, {Symbol: "BCH", SymbolName: "Bitcoin Cash"}, {Symbol: "UNRANKED", SymbolName: "Unranked Bitcoin"},
		}},
		{name: "US stocks in search order", category: usStockCategory, listings: usStockListings, typedText: "SPDR S&P 500", expectedCandidates: []dto.SymbolCandidateDto{
			{Symbol: "SPY", SymbolName: "State Street SPDR S&P 500 ETF Trust"}, {Symbol: "SPYX", SymbolName: "State Street SPDR S&P 500 Fossil Fuel Reserves Free ETF"}, {Symbol: "EFIV", SymbolName: "State Street SPDR S&P 500 ESG ETF"},
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := identify(t, testCase.category, testCase.listings, testCase.typedText)

			symbolAmbiguousError, isAmbiguous := errors.AsType[*domains.SymbolAmbiguousError](err)
			require.True(t, isAmbiguous)
			assert.ErrorIs(t, err, domains.ErrSymbolAmbiguous)
			assert.Equal(t, "符合多個標的，請改填代號或更完整的名稱", err.Error())
			assert.Equal(t, testCase.expectedCandidates, symbolAmbiguousError.Candidates)
		})
	}
}

func TestSymbolDirectoryDomain_ListsAtMostTenTaiwanCandidatesBySymbol(t *testing.T) {
	listings := []vo.SymbolListingVo{}
	for index := 12; index >= 1; index-- {
		listings = append(listings, vo.SymbolListingVo{Symbol: fmt.Sprintf("28%02d", index), Name: fmt.Sprintf("國泰%d號", index)})
	}

	_, err := identify(t, twStockCategory, listings, "國泰")

	symbolAmbiguousError, isAmbiguous := errors.AsType[*domains.SymbolAmbiguousError](err)
	require.True(t, isAmbiguous)
	require.Len(t, symbolAmbiguousError.Candidates, 10)
	assert.Equal(t, dto.SymbolCandidateDto{Symbol: "2801", SymbolName: "國泰1號"}, symbolAmbiguousError.Candidates[0])
	assert.Equal(t, dto.SymbolCandidateDto{Symbol: "2810", SymbolName: "國泰10號"}, symbolAmbiguousError.Candidates[9])
}

func TestSymbolDirectoryDomain_RejectsWhatNoListingMatches(t *testing.T) {
	testCases := []struct {
		name      string
		category  vo.MarketCategoryVo
		listings  []vo.SymbolListingVo
		typedText string
	}{
		{name: "unknown Taiwan name", category: twStockCategory, listings: twStockListings, typedText: "台基電"},
		{name: "unknown coin", category: cryptoCategory, listings: cryptoListings, typedText: "ZZZQQ"},
		{name: "unknown US symbol", category: usStockCategory, listings: usStockListings, typedText: "ZZZZ"},
		{name: "empty directory", category: usStockCategory, listings: []vo.SymbolListingVo{}, typedText: "Tesla"},
		{name: "listing without a name is ignored", category: twStockCategory, listings: []vo.SymbolListingVo{{Symbol: "9999", Name: " "}}, typedText: "9999"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := identify(t, testCase.category, testCase.listings, testCase.typedText)

			assert.ErrorIs(t, err, domains.ErrSymbolNotFound)
		})
	}
}

func TestSymbolDirectoryDomain_FindsOnlyByExactSymbol(t *testing.T) {
	testCases := []struct {
		name           string
		category       vo.MarketCategoryVo
		listings       []vo.SymbolListingVo
		symbol         string
		expectedSymbol string
		expectedName   string
		expectedError  error
	}{
		{name: "listed ETF", category: twStockCategory, listings: twStockListings, symbol: "0050", expectedSymbol: "0050", expectedName: "元大台灣50"},
		{name: "best ranked coin of a shared symbol", category: cryptoCategory, listings: append([]vo.SymbolListingVo{{Symbol: "BTC", Name: "Bitcoin Fork", Priority: 900}}, cryptoListings...), symbol: "BTC", expectedSymbol: "BTC", expectedName: "Bitcoin"},
		{name: "a name is not a symbol", category: twStockCategory, listings: twStockListings, symbol: "台積電", expectedError: domains.ErrSymbolNotFound},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			symbol, err := vo.NewSymbolVo(testCase.symbol, testCase.category)
			require.NoError(t, err)

			resolvedSymbol, err := domains.NewSymbolDirectoryDomain(testCase.category, testCase.listings).FindBySymbol(symbol)

			assert.ErrorIs(t, err, testCase.expectedError)
			assert.Equal(t, testCase.expectedSymbol, resolvedSymbol.Symbol.Value)
			assert.Equal(t, testCase.expectedName, resolvedSymbol.SymbolName.Value)
		})
	}
}
