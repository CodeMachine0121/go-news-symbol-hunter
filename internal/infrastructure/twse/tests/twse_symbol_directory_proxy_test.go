package twse_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/twse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const listedDailyClosings = `[{"Date":"1151028","Code":"2330","Name":"台積電","ClosingPrice":"2575.00"},{"Date":"1151028","Code":" 0050 ","Name":" 元大台灣50 ","ClosingPrice":"190.00"},{"Date":"1151028","Code":"","Name":"沒有代號","ClosingPrice":"1.00"}]`

var lookedUpAt = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func startListedDailyClosingServer(t *testing.T, statusCode int, body string) (*httptest.Server, *atomic.Int32) {
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, requestCount
}

func TestTwseSymbolDirectoryProxy_ListsStocksAndEtfsFromTheDailyClosingQuotes(t *testing.T) {
	server, _ := startListedDailyClosingServer(t, http.StatusOK, listedDailyClosings)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)

	listings, err := twse.NewTwseOpenDataProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL).SearchSymbolListings(context.Background(), "台積電")

	require.NoError(t, err)
	assert.ElementsMatch(t, []vo.SymbolListingVo{{Symbol: "2330", Name: "台積電"}, {Symbol: "0050", Name: "元大台灣50"}}, listings)
}

func TestTwseSymbolDirectoryProxy_SharesOneDownloadWithClosingPrices(t *testing.T) {
	server, requestCount := startListedDailyClosingServer(t, http.StatusOK, listedDailyClosings)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	twseOpenDataProxy := twse.NewTwseOpenDataProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	_, listingError := twseOpenDataProxy.SearchSymbolListings(context.Background(), "0050")
	_, priceError := twseOpenDataProxy.FetchPrice(context.Background(), "0050")

	require.NoError(t, listingError)
	require.NoError(t, priceError)
	assert.Equal(t, int32(1), requestCount.Load())
}

func TestTwseSymbolDirectoryProxy_ReportsUnavailableMalformedOrEmptyDirectories(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unavailable", statusCode: http.StatusBadGateway},
		{name: "malformed", statusCode: http.StatusOK, body: "{"},
		{name: "empty", statusCode: http.StatusOK, body: `[]`},
		{name: "without codes", statusCode: http.StatusOK, body: `[{"Code":"","Name":"沒有代號"}]`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server, _ := startListedDailyClosingServer(t, testCase.statusCode, testCase.body)
			clockProxy := mocks.NewMockIClockProxy(t)
			clockProxy.EXPECT().Now().Return(lookedUpAt)

			listings, err := twse.NewTwseOpenDataProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL).SearchSymbolListings(context.Background(), "2330")

			assert.Error(t, err)
			assert.Empty(t, listings)
		})
	}
}
