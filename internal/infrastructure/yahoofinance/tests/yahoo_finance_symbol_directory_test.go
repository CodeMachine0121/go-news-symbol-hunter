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
		{"symbol":"TSLT","shortname":" T-REX 2X Long Tesla Daily Targe ","quoteType":"ETF","exchange":"BTS"},
		{"symbol":"BRK-B","shortname":"Berkshire Hathaway Inc. New","longname":"Berkshire Hathaway Inc.","quoteType":"EQUITY","exchange":"NYQ"}
	]}`)

	require.NoError(t, err)
	assert.Equal(t, "/v1/finance/search?newsCount=0&q=Tesla&quotesCount=20", receivedRequest)
	assert.Equal(t, []vo.SymbolListingVo{
		{Symbol: "TSLA", Name: "Tesla, Inc.", Priority: 0},
		{Symbol: "TSLT", Name: "T-REX 2X Long Tesla Daily Targe", Priority: 1},
		{Symbol: "BRK.B", Name: "Berkshire Hathaway Inc.", Priority: 2},
	}, listings)
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
