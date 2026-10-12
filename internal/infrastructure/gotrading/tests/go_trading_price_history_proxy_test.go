package gotrading_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/gotrading"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var taipei = time.FixedZone("Asia/Taipei", 8*60*60)

var minuteQuery = vo.PriceHistoryQueryVo{
	Symbol:   "2330",
	Category: "twStock",
	Interval: "1m",
	Since:    time.Date(2026, 10, 6, 18, 15, 0, 0, taipei),
	Until:    time.Date(2026, 10, 7, 10, 15, 0, 0, taipei),
}

// stands in for go-trading over real HTTP: every sign-in issues a new token, and the series only answers to the newest one
type goTradingServer struct {
	server           *httptest.Server
	signInCount      atomic.Int32
	validToken       atomic.Value
	receivedQueries  chan url.Values
	receivedSignIns  chan map[string]string
	signInStatus     int
	signInBody       string
	truncateSeries   bool
	seriesStatus     int
	seriesBody       string
	rejectEveryToken bool
}

func startGoTradingServer(t *testing.T) *goTradingServer {
	goTrading := &goTradingServer{
		receivedQueries: make(chan url.Values, 10),
		receivedSignIns: make(chan map[string]string, 10),
		signInStatus:    http.StatusOK,
		seriesStatus:    http.StatusOK,
		seriesBody:      `{"symbol":"2330","interval":"1m","kCandles":[{"openTime":"2026-10-07T02:14:00Z","close":"1000.5","open":"999"},{"openTime":"2026-10-07T02:15:00Z","close":"1001"}]}`,
	}
	goTrading.validToken.Store("")
	goTrading.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/sessions":
			signIn := map[string]string{}
			_ = json.NewDecoder(request.Body).Decode(&signIn)
			goTrading.receivedSignIns <- signIn
			if goTrading.signInStatus != http.StatusOK {
				writer.WriteHeader(goTrading.signInStatus)
				return
			}
			if goTrading.signInBody != "" {
				_, _ = writer.Write([]byte(goTrading.signInBody))
				return
			}
			token := fmt.Sprintf("token-%d", goTrading.signInCount.Add(1))
			goTrading.validToken.Store(token)
			_, _ = writer.Write([]byte(`{"accessToken":"` + token + `","refreshToken":"unused"}`))
		case "/k-candles/series":
			if goTrading.rejectEveryToken || request.Header.Get("Authorization") != "Bearer "+goTrading.validToken.Load().(string) {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			goTrading.receivedQueries <- request.URL.Query()
			if goTrading.truncateSeries {
				// promises more than it sends, then drops the connection
				writer.Header().Set("Content-Length", "1000")
				_, _ = writer.Write([]byte(`{"kCandles":`))
				writer.(http.Flusher).Flush()
				panic(http.ErrAbortHandler)
			}
			writer.WriteHeader(goTrading.seriesStatus)
			_, _ = writer.Write([]byte(goTrading.seriesBody))
		}
	}))
	t.Cleanup(goTrading.server.Close)
	return goTrading
}

func (goTrading *goTradingServer) proxy() *gotrading.GoTradingPriceHistoryProxy {
	return gotrading.NewGoTradingPriceHistoryProxy(goTrading.server.Client(), gotrading.GoTradingAccount{BaseUrl: goTrading.server.URL, Email: "owner@example.com", Password: "secret"})
}

func TestGoTradingPriceHistoryProxy_ReadsTheSeriesAfterSigningIn(t *testing.T) {
	goTrading := startGoTradingServer(t)

	priceCandles, err := goTrading.proxy().FetchCandles(context.Background(), minuteQuery)

	require.NoError(t, err)
	assert.Equal(t, []vo.PriceCandleVo{
		{OpenTime: time.Date(2026, 10, 7, 2, 14, 0, 0, time.UTC), Close: decimal.RequireFromString("1000.5")},
		{OpenTime: time.Date(2026, 10, 7, 2, 15, 0, 0, time.UTC), Close: decimal.RequireFromString("1001")},
	}, priceCandles)
	assert.Equal(t, map[string]string{"email": "owner@example.com", "password": "secret"}, <-goTrading.receivedSignIns)
	assert.Equal(t, url.Values{
		"symbol":    {"2330"},
		"interval":  {"1m"},
		"startTime": {"2026-10-06T10:15:00Z"},
		"endTime":   {"2026-10-07T02:15:00Z"},
	}, <-goTrading.receivedQueries)
}

func TestGoTradingPriceHistoryProxy_NamesCryptoByItsUsdtPair(t *testing.T) {
	goTrading := startGoTradingServer(t)

	_, err := goTrading.proxy().FetchCandles(context.Background(), vo.PriceHistoryQueryVo{Symbol: "BTC", Category: "crypto", Interval: "1d"})

	require.NoError(t, err)
	assert.Equal(t, "BTCUSDT", (<-goTrading.receivedQueries).Get("symbol"))
}

func TestGoTradingPriceHistoryProxy_AnUnknownSymbolHasNoCandles(t *testing.T) {
	goTrading := startGoTradingServer(t)
	goTrading.seriesBody = `{"symbol":"6488","interval":"1m","kCandles":[]}`

	priceCandles, err := goTrading.proxy().FetchCandles(context.Background(), minuteQuery)

	require.NoError(t, err)
	assert.Empty(t, priceCandles)
}

func TestGoTradingPriceHistoryProxy_SignsInOnceForManyFetches(t *testing.T) {
	goTrading := startGoTradingServer(t)
	goTradingProxy := goTrading.proxy()
	var waitGroup sync.WaitGroup
	for range 5 {
		waitGroup.Go(func() {
			_, err := goTradingProxy.FetchCandles(context.Background(), minuteQuery)
			assert.NoError(t, err)
		})
	}
	waitGroup.Wait()

	assert.Equal(t, int32(1), goTrading.signInCount.Load())
}

func TestGoTradingPriceHistoryProxy_SignsInAgainWhenTheTokenIsRejected(t *testing.T) {
	goTrading := startGoTradingServer(t)
	goTradingProxy := goTrading.proxy()
	_, err := goTradingProxy.FetchCandles(context.Background(), minuteQuery)
	require.NoError(t, err)
	// go-trading forgets the token, as it does once a token expires
	goTrading.validToken.Store("expired")

	priceCandles, err := goTradingProxy.FetchCandles(context.Background(), minuteQuery)

	require.NoError(t, err)
	assert.Len(t, priceCandles, 2)
	assert.Equal(t, int32(2), goTrading.signInCount.Load())
}

func TestGoTradingPriceHistoryProxy_FailsWhenGoTradingCannotAnswer(t *testing.T) {
	testCases := []struct {
		name  string
		given func(goTrading *goTradingServer)
	}{
		{name: "sign-in refused", given: func(goTrading *goTradingServer) { goTrading.signInStatus = http.StatusUnauthorized }},
		{name: "sign-in refused with a body", given: func(goTrading *goTradingServer) {
			goTrading.signInBody = `{"error":{"code":"invalid_credentials"}}`
		}},
		{name: "sign-in unreadable", given: func(goTrading *goTradingServer) { goTrading.signInBody = `{` }},
		{name: "series cut off", given: func(goTrading *goTradingServer) { goTrading.truncateSeries = true }},
		{name: "every token rejected", given: func(goTrading *goTradingServer) { goTrading.rejectEveryToken = true }},
		{name: "series failed", given: func(goTrading *goTradingServer) { goTrading.seriesStatus = http.StatusBadGateway }},
		{name: "series unreadable", given: func(goTrading *goTradingServer) { goTrading.seriesBody = `{` }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			goTrading := startGoTradingServer(t)
			testCase.given(goTrading)

			_, err := goTrading.proxy().FetchCandles(context.Background(), minuteQuery)

			assert.Error(t, err)
		})
	}
}

func TestGoTradingPriceHistoryProxy_FailsWhenGoTradingIsUnreachable(t *testing.T) {
	goTrading := startGoTradingServer(t)
	goTradingProxy := goTrading.proxy()
	goTrading.server.Close()

	_, err := goTradingProxy.FetchCandles(context.Background(), minuteQuery)

	assert.Error(t, err)
}

func TestGoTradingPriceHistoryProxy_FailsWithoutAnAccount(t *testing.T) {
	goTrading := startGoTradingServer(t)
	testCases := []gotrading.GoTradingAccount{
		{Email: "owner@example.com", Password: "secret"},
		{BaseUrl: goTrading.server.URL, Password: "secret"},
		{BaseUrl: goTrading.server.URL, Email: "owner@example.com"},
	}
	for _, account := range testCases {
		_, err := gotrading.NewGoTradingPriceHistoryProxy(goTrading.server.Client(), account).FetchCandles(context.Background(), minuteQuery)

		assert.Error(t, err)
	}
	assert.Equal(t, int32(0), goTrading.signInCount.Load())
}

func TestGoTradingPriceHistoryProxy_FailsWhenGoTradingGoesAwayAfterSigningIn(t *testing.T) {
	goTrading := startGoTradingServer(t)
	goTradingProxy := goTrading.proxy()
	_, err := goTradingProxy.FetchCandles(context.Background(), minuteQuery)
	require.NoError(t, err)
	goTrading.server.Close()

	_, err = goTradingProxy.FetchCandles(context.Background(), minuteQuery)

	assert.Error(t, err)
}

func TestGoTradingPriceHistoryProxy_FailsWithAnUnusableAddress(t *testing.T) {
	goTradingProxy := gotrading.NewGoTradingPriceHistoryProxy(http.DefaultClient, gotrading.GoTradingAccount{BaseUrl: "http://bad host", Email: "owner@example.com", Password: "secret"})

	_, err := goTradingProxy.FetchCandles(context.Background(), minuteQuery)

	assert.Error(t, err)
}

func TestGoTradingPriceHistoryProxy_DoesNotKeepSigningInWhenNoTokenIsIssued(t *testing.T) {
	goTrading := startGoTradingServer(t)
	goTrading.signInBody = `{"error":{"code":"invalid_credentials"}}`

	_, err := goTrading.proxy().FetchCandles(context.Background(), minuteQuery)

	assert.Error(t, err)
	assert.Len(t, goTrading.receivedSignIns, 1)
}
