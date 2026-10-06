package coingecko_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/coingecko"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
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

func TestCoinGeckoCryptocurrencyProxy_PicksTheBestRankedExactSymbolMatch(t *testing.T) {
	testCases := []struct {
		name             string
		body             string
		expectedCoinName string
		expectedFound    bool
	}{
		{name: "best market cap rank wins among exact symbol matches", body: `{"coins":[{"symbol":"BTCP","name":"Bitcoin Pro","market_cap_rank":805},{"symbol":"btc","name":"Bitcoin Fork","market_cap_rank":900},{"symbol":"BTC","name":"Bitcoin","market_cap_rank":1}]}`, expectedCoinName: "Bitcoin", expectedFound: true},
		{name: "ranked match beats an unranked one", body: `{"coins":[{"symbol":"BTC","name":"Unranked Bitcoin","market_cap_rank":null},{"symbol":"BTC","name":"Bitcoin","market_cap_rank":3}]}`, expectedCoinName: "Bitcoin", expectedFound: true},
		{name: "unranked match does not replace a ranked one", body: `{"coins":[{"symbol":"BTC","name":"Bitcoin","market_cap_rank":3},{"symbol":"BTC","name":"Unranked Bitcoin"}]}`, expectedCoinName: "Bitcoin", expectedFound: true},
		{name: "first unranked match is kept when none is ranked", body: `{"coins":[{"symbol":"BTC","name":"First"},{"symbol":"BTC","name":"Second"}]}`, expectedCoinName: "First", expectedFound: true},
		{name: "no exact symbol match", body: `{"coins":[{"symbol":"BTCP","name":"Bitcoin Pro","market_cap_rank":805}]}`, expectedFound: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server, _, receivedQuery := startSearchServer(t, http.StatusOK, testCase.body)
			clockProxy := mocks.NewMockIClockProxy(t)
			clockProxy.EXPECT().Now().Return(lookedUpAt)

			coinName, found, err := coingecko.NewCoinGeckoCryptocurrencyProxy(utilities.NewHttpBodyReader(server.Client()), clockProxy, server.URL).FindCoinName("BTC")

			require.NoError(t, err)
			assert.Equal(t, "BTC", *receivedQuery)
			assert.Equal(t, testCase.expectedFound, found)
			assert.Equal(t, testCase.expectedCoinName, coinName)
		})
	}
}

func TestCoinGeckoCryptocurrencyProxy_CachesLookupsIncludingMissesForTwentyFourHours(t *testing.T) {
	server, requestCount, _ := startSearchServer(t, http.StatusOK, `{"coins":[]}`)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(24*time.Hour - time.Second)).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(24 * time.Hour)).Once()
	coinGeckoCryptocurrencyProxy := coingecko.NewCoinGeckoCryptocurrencyProxy(utilities.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	_, firstFound, _ := coinGeckoCryptocurrencyProxy.FindCoinName("NOTACOIN")
	_, cachedFound, _ := coinGeckoCryptocurrencyProxy.FindCoinName("NOTACOIN")
	assert.Equal(t, int32(1), requestCount.Load())
	_, _, _ = coinGeckoCryptocurrencyProxy.FindCoinName("NOTACOIN")

	assert.False(t, firstFound)
	assert.False(t, cachedFound)
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

			_, found, err := coingecko.NewCoinGeckoCryptocurrencyProxy(utilities.NewHttpBodyReader(server.Client()), clockProxy, server.URL).FindCoinName("BTC")

			assert.Error(t, err)
			assert.False(t, found)
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
	coinGeckoCryptocurrencyProxy := coingecko.NewCoinGeckoCryptocurrencyProxy(utilities.NewHttpBodyReader(server.Client()), clockProxy, server.URL)
	slowLookupDone := make(chan struct{})
	go func() {
		_, _, _ = coinGeckoCryptocurrencyProxy.FindCoinName("SLOW")
		close(slowLookupDone)
	}()
	time.Sleep(50 * time.Millisecond)

	coinName, found, err := coinGeckoCryptocurrencyProxy.FindCoinName("ETH")

	close(releaseSlowLookup)
	<-slowLookupDone
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Ethereum", coinName)
}
