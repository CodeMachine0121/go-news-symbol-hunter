package coingecko_test

import (
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/coingecko"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var lookedUpAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func startSearchServer(t *testing.T, statusCode int, body string) (*httptest.Server, *atomic.Int32, *string) {
	requestCount := &atomic.Int32{}
	receivedQuery := new(string)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		*receivedQuery = request.URL.Query().Get("query")
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, requestCount, receivedQuery
}

func TestCoinGeckoCryptocurrencyProxy_ListsEverySearchResultRankedByMarketCap(t *testing.T) {
	server, _, receivedQuery := startSearchServer(t, http.StatusOK, `{"coins":[{"symbol":"BTC","name":"Bitcoin","market_cap_rank":1},{"symbol":"BITCOIN","name":"HarryPotterObamaSonic10Inu","market_cap_rank":500},{"symbol":"BTCX","name":"Unranked Bitcoin","market_cap_rank":null}]}`)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)

	listings, err := coingecko.NewCoinGeckoCryptocurrencyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL).SearchSymbolListings(context.Background(), " Bitcoin ")

	require.NoError(t, err)
	assert.Equal(t, "Bitcoin", *receivedQuery)
	assert.Equal(t, []vo.SymbolListingVo{
		{Symbol: "BTC", Name: "Bitcoin", Priority: 1},
		{Symbol: "BITCOIN", Name: "HarryPotterObamaSonic10Inu", Priority: 500},
		{Symbol: "BTCX", Name: "Unranked Bitcoin", Priority: math.MaxInt},
	}, listings)
}

func TestCoinGeckoCryptocurrencyProxy_SharesTheCacheAcrossLetterCases(t *testing.T) {
	server, requestCount, _ := startSearchServer(t, http.StatusOK, `{"coins":[{"symbol":"btc","name":"Bitcoin","market_cap_rank":1}]}`)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	coinGeckoCryptocurrencyProxy := coingecko.NewCoinGeckoCryptocurrencyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "BITCOIN")
	cachedListings, err := coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "bitcoin")

	require.NoError(t, err)
	assert.Equal(t, []vo.SymbolListingVo{{Symbol: "btc", Name: "Bitcoin", Priority: 1}}, cachedListings)
	assert.Equal(t, int32(1), requestCount.Load())
}

func TestCoinGeckoCryptocurrencyProxy_CachesLookupsIncludingMissesForTwentyFourHours(t *testing.T) {
	server, requestCount, _ := startSearchServer(t, http.StatusOK, `{"coins":[]}`)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(24*time.Hour - time.Second)).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(24 * time.Hour)).Once()
	coinGeckoCryptocurrencyProxy := coingecko.NewCoinGeckoCryptocurrencyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	firstListings, _ := coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "NOTACOIN")
	cachedListings, _ := coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "NOTACOIN")
	assert.Equal(t, int32(1), requestCount.Load())
	_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "NOTACOIN")

	assert.Empty(t, firstListings)
	assert.Empty(t, cachedListings)
	assert.Equal(t, int32(2), requestCount.Load())
}

func TestCoinGeckoCryptocurrencyProxy_ReportsUnavailableOrMalformedResponses(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "rate limited", statusCode: http.StatusTooManyRequests},
		{name: "malformed", statusCode: http.StatusOK, body: "{"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server, _, _ := startSearchServer(t, testCase.statusCode, testCase.body)
			clockProxy := mocks.NewMockIClockProxy(t)
			clockProxy.EXPECT().Now().Return(lookedUpAt)

			listings, err := coingecko.NewCoinGeckoCryptocurrencyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL).SearchSymbolListings(context.Background(), "BTC")

			assert.Error(t, err)
			assert.Empty(t, listings)
		})
	}
}

func TestCoinGeckoCryptocurrencyProxy_DoesNotQueueOtherSymbolsBehindASlowLookup(t *testing.T) {
	releaseSlowLookup := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("query") == "SLOW" {
			<-releaseSlowLookup
		}
		_, _ = writer.Write([]byte(`{"coins":[{"symbol":"ETH","name":"Ethereum","market_cap_rank":2}]}`))
	}))
	defer server.Close()
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	coinGeckoCryptocurrencyProxy := coingecko.NewCoinGeckoCryptocurrencyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)
	slowLookupDone := make(chan struct{})
	go func() {
		_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "SLOW")
		close(slowLookupDone)
	}()
	time.Sleep(50 * time.Millisecond)

	listings, err := coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "ETH")

	close(releaseSlowLookup)
	<-slowLookupDone
	require.NoError(t, err)
	assert.Equal(t, []vo.SymbolListingVo{{Symbol: "ETH", Name: "Ethereum", Priority: 2}}, listings)
}

func TestCoinGeckoCryptocurrencyProxy_KeepsTheCacheBounded(t *testing.T) {
	server, requestCount, _ := startSearchServer(t, http.StatusOK, `{"coins":[]}`)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	coinGeckoCryptocurrencyProxy := coingecko.NewCoinGeckoCryptocurrencyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)
	for index := range 1001 {
		_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), fmt.Sprintf("COIN%d", index))
	}

	_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "COIN0")
	_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "COIN1000")

	assert.Equal(t, int32(1002), requestCount.Load())
}

func TestCoinGeckoCryptocurrencyProxy_EvictsExpiredLookupsWhenFull(t *testing.T) {
	server, requestCount, _ := startSearchServer(t, http.StatusOK, `{"coins":[]}`)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Times(1000)
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(25 * time.Hour))
	coinGeckoCryptocurrencyProxy := coingecko.NewCoinGeckoCryptocurrencyProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)
	for index := range 1000 {
		_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), fmt.Sprintf("COIN%d", index))
	}

	_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "FRESH")
	_, _ = coinGeckoCryptocurrencyProxy.SearchSymbolListings(context.Background(), "FRESH")

	assert.Equal(t, int32(1001), requestCount.Load())
}
