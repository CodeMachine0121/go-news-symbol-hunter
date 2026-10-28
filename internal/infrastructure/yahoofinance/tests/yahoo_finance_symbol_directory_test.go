package yahoofinance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/yahoofinance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchListingsFrom(t *testing.T, query string, statusCode int, body string) ([]vo.SymbolListingVo, string, error) {
	receivedRequest := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedRequest = request.URL.Path + "?" + request.URL.RawQuery
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	listings, err := yahoofinance.NewYahooFinanceProxy(httpfetch.NewHttpBodyReader(server.Client()), nil, yahoofinance.YahooFinanceUrls{Search: server.URL + "/v1/finance/search"}).SearchSymbolListings(context.Background(), query)
	return listings, receivedRequest, err
}

func TestYahooFinanceProxy_ListsOnlyUsStocksAndEtfsInSearchOrder(t *testing.T) {
	listings, receivedRequest, err := searchListingsFrom(t, "Tesla", http.StatusOK, `{"quotes":[
		{"symbol":"TSLA","shortname":"Tesla, Inc.","longname":"Tesla, Inc.","quoteType":"EQUITY","exchange":"NMS"},
		{"symbol":"STSLA=F","shortname":"Tesla Inc Single Stock futures,","quoteType":"FUTURE","exchange":"CME"},
		{"symbol":"TSLA.TO","shortname":"TESLA CDR (CAD HEDGED)","longname":"Tesla, Inc.","quoteType":"EQUITY","exchange":"TOR"},
		{"symbol":"FXAIX","shortname":"Fidelity 500 Index","quoteType":"MUTUALFUND","exchange":"NAS"},
		{"symbol":"TSLT","shortname":" T-REX 2X Long Tesla Daily Targe ","quoteType":"ETF","exchange":"BTS"},
		{"symbol":"TCEHY","longname":"Tencent Holdings Limited","quoteType":"EQUITY","exchange":"PNK"},
		{"symbol":"BRK-B","shortname":"Berkshire Hathaway Inc. New","longname":"Berkshire Hathaway Inc.","quoteType":"EQUITY","exchange":"NYQ"}
	]}`)

	require.NoError(t, err)
	assert.Equal(t, "/v1/finance/search?newsCount=0&q=Tesla&quotesCount=20", receivedRequest)
	assert.Equal(t, []vo.SymbolListingVo{
		{Symbol: "TSLA", Name: "Tesla, Inc.", Priority: 0},
		{Symbol: "TSLT", Name: "T-REX 2X Long Tesla Daily Targe", Priority: 1},
		{Symbol: "TCEHY", Name: "Tencent Holdings Limited", Priority: 2},
		{Symbol: "BRK.B", Name: "Berkshire Hathaway Inc.", Priority: 3},
	}, listings)
}

func TestYahooFinanceProxy_SearchesShareClassesInYahooForm(t *testing.T) {
	testCases := []struct {
		query           string
		expectedRequest string
	}{
		{query: "BRK.B", expectedRequest: "/v1/finance/search?newsCount=0&q=BRK-B&quotesCount=20"},
		{query: "Apple Inc.", expectedRequest: "/v1/finance/search?newsCount=0&q=Apple+Inc.&quotesCount=20"},
		{query: "AMAZON.COM", expectedRequest: "/v1/finance/search?newsCount=0&q=AMAZON.COM&quotesCount=20"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.query, func(t *testing.T) {
			_, receivedRequest, err := searchListingsFrom(t, testCase.query, http.StatusOK, `{"quotes":[]}`)

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedRequest, receivedRequest)
		})
	}
}

func TestYahooFinanceProxy_FailsToSearchWhenTheDirectoryCannotBeRead(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unavailable", statusCode: http.StatusTooManyRequests},
		{name: "malformed", statusCode: http.StatusOK, body: `{`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			listings, _, err := searchListingsFrom(t, "Tesla", testCase.statusCode, testCase.body)

			assert.Error(t, err)
			assert.Empty(t, listings)
		})
	}
}
