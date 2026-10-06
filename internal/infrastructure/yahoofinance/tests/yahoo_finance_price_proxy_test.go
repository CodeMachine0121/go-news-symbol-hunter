package yahoofinance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/yahoofinance"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fetchFrom(t *testing.T, statusCode int, body string) (vo.PriceQuoteVo, string, error) {
	return fetchSymbolFrom(t, "AAPL", statusCode, body)
}

func fetchSymbolFrom(t *testing.T, symbol string, statusCode int, body string) (vo.PriceQuoteVo, string, error) {
	receivedPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedPath = request.URL.Path + "?" + request.URL.RawQuery
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	priceQuote, err := yahoofinance.NewYahooFinanceProxy(httpfetch.NewHttpBodyReader(server.Client()), nil, yahoofinance.YahooFinanceUrls{Chart: server.URL + "/v8/finance/chart/"}).FetchPrice(context.Background(), symbol)
	return priceQuote, receivedPath, err
}

func TestYahooFinancePriceProxy_QuotesTheRegularMarketPrice(t *testing.T) {
	priceQuote, receivedPath, err := fetchFrom(t, http.StatusOK, `{"chart":{"result":[{"meta":{"symbol":"AAPL","currency":"USD","regularMarketPrice":332.94,"regularMarketTime":1791326022}}],"error":null}}`)

	require.NoError(t, err)
	assert.Equal(t, "/v8/finance/chart/AAPL?range=1d&interval=1d", receivedPath)
	assert.Equal(t, vo.PriceQuoteVo{Price: decimal.RequireFromString("332.94"), Currency: "USD", PricedAt: time.Date(2026, 10, 6, 22, 33, 42, 0, time.UTC), Source: "Yahoo 財經"}, priceQuote)
	assert.Equal(t, "332.94", priceQuote.Price.String())
}

func TestYahooFinancePriceProxy_KeepsPrecisionBeyondFloatingPoint(t *testing.T) {
	priceQuote, _, err := fetchFrom(t, http.StatusOK, `{"chart":{"result":[{"meta":{"currency":"USD","regularMarketPrice":123456789.123456789,"regularMarketTime":1791326022}}]}}`)

	require.NoError(t, err)
	assert.Equal(t, "123456789.123456789", priceQuote.Price.String())
}

func TestYahooFinancePriceProxy_FailsWithoutAUsablePrice(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unknown symbol", statusCode: http.StatusNotFound, body: `{"chart":{"result":null,"error":{"code":"Not Found"}}}`},
		{name: "no result", statusCode: http.StatusOK, body: `{"chart":{"result":[],"error":null}}`},
		{name: "no quote time", statusCode: http.StatusOK, body: `{"chart":{"result":[{"meta":{"currency":"USD","regularMarketPrice":1}}]}}`},
		{name: "malformed", statusCode: http.StatusOK, body: `{`},
		{name: "zero price", statusCode: http.StatusOK, body: `{"chart":{"result":[{"meta":{"currency":"USD","regularMarketPrice":0,"regularMarketTime":1791299622}}]}}`},
		{name: "missing currency", statusCode: http.StatusOK, body: `{"chart":{"result":[{"meta":{"currency":null,"regularMarketPrice":1,"regularMarketTime":1791299622}}]}}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := fetchFrom(t, testCase.statusCode, testCase.body)

			assert.Error(t, err)
		})
	}
}

func TestYahooFinanceProxy_WritesShareClassesWithADash(t *testing.T) {
	_, receivedPath, err := fetchSymbolFrom(t, "BRK.B", http.StatusOK, `{"chart":{"result":[{"meta":{"currency":"USD","regularMarketPrice":500,"regularMarketTime":1791326022}}]}}`)

	require.NoError(t, err)
	assert.Equal(t, "/v8/finance/chart/BRK-B?range=1d&interval=1d", receivedPath)
}
