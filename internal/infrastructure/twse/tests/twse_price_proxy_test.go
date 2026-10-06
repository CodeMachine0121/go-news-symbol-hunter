package twse_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/twse"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dailyClosings = `[{"Date":"1151005","Code":"2330","Name":"台積電","ClosingPrice":"2575.00"},{"Date":"1151005","Code":"1101","ClosingPrice":"1,234.50"},{"Date":"1151005","Code":"9998","ClosingPrice":""},{"Date":"1151005","Code":"9997","ClosingPrice":"0.00"},{"Date":"115","Code":"9996","ClosingPrice":"10"},{"Date":"ABC1005","Code":"9995","ClosingPrice":"10"}]`

func startDailyClosingServer(t *testing.T, statusCode int, body string) (*httptest.Server, *atomic.Int32) {
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, requestCount
}

func TestTwsePriceProxy_QuotesTheLatestClosingPrice(t *testing.T) {
	server, _ := startDailyClosingServer(t, http.StatusOK, dailyClosings)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	twsePriceProxy := twse.NewTwsePriceProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	priceQuote, err := twsePriceProxy.FetchPrice(context.Background(), "2330")
	thousandsPriceQuote, thousandsError := twsePriceProxy.FetchPrice(context.Background(), "1101")

	require.NoError(t, err)
	assert.Equal(t, "2575", priceQuote.Price.String())
	assert.True(t, decimal.RequireFromString("2575.00").Equal(priceQuote.Price))
	assert.Equal(t, "TWD", priceQuote.Currency)
	assert.Equal(t, "證交所", priceQuote.Source)
	assert.True(t, priceQuote.PricedAt.Equal(time.Date(2026, 10, 4, 16, 0, 0, 0, time.UTC)))
	require.NoError(t, thousandsError)
	assert.Equal(t, "1234.5", thousandsPriceQuote.Price.String())
}

func TestTwsePriceProxy_FailsWithoutAUsablePrice(t *testing.T) {
	server, _ := startDailyClosingServer(t, http.StatusOK, dailyClosings)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt)
	twsePriceProxy := twse.NewTwsePriceProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	for _, stockCode := range []string{"9999", "9998", "9997", "9996", "9995"} {
		_, err := twsePriceProxy.FetchPrice(context.Background(), stockCode)

		assert.Error(t, err, stockCode)
	}
}

func TestTwsePriceProxy_ReusesTheDailyClosingsForAnHour(t *testing.T) {
	server, requestCount := startDailyClosingServer(t, http.StatusOK, dailyClosings)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(time.Hour - time.Second)).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(time.Hour)).Once()
	twsePriceProxy := twse.NewTwsePriceProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	_, _ = twsePriceProxy.FetchPrice(context.Background(), "2330")
	_, _ = twsePriceProxy.FetchPrice(context.Background(), "2330")
	assert.Equal(t, int32(1), requestCount.Load())
	_, _ = twsePriceProxy.FetchPrice(context.Background(), "2330")

	assert.Equal(t, int32(2), requestCount.Load())
}

func TestTwsePriceProxy_ReportsUnavailableOrEmptyData(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unavailable", statusCode: http.StatusBadGateway},
		{name: "malformed", statusCode: http.StatusOK, body: `{`},
		{name: "empty", statusCode: http.StatusOK, body: `[]`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server, _ := startDailyClosingServer(t, testCase.statusCode, testCase.body)
			clockProxy := mocks.NewMockIClockProxy(t)
			clockProxy.EXPECT().Now().Return(lookedUpAt)

			_, err := twse.NewTwsePriceProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL).FetchPrice(context.Background(), "2330")

			assert.Error(t, err)
		})
	}
}
