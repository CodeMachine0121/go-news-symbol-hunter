package binance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/binance"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pricedAt = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func fetchFrom(t *testing.T, statusCode int, body string) (vo.PriceQuoteVo, string, error) {
	receivedSymbol := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedSymbol = request.URL.Query().Get("symbol")
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(pricedAt).Maybe()

	priceQuote, err := binance.NewBinancePriceProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL).FetchPrice(context.Background(), "BTC")
	return priceQuote, receivedSymbol, err
}

func TestBinancePriceProxy_QuotesTheUsdtPairExactly(t *testing.T) {
	priceQuote, receivedSymbol, err := fetchFrom(t, http.StatusOK, `{"symbol":"BTCUSDT","price":"0.00001234"}`)

	require.NoError(t, err)
	assert.Equal(t, "BTCUSDT", receivedSymbol)
	assert.Equal(t, vo.PriceQuoteVo{Price: decimal.RequireFromString("0.00001234"), Currency: "USDT", PricedAt: pricedAt, Source: "Binance"}, priceQuote)
	assert.Equal(t, "0.00001234", priceQuote.Price.String())
}

func TestBinancePriceProxy_KeepsPrecisionBeyondFloatingPoint(t *testing.T) {
	priceQuote, _, err := fetchFrom(t, http.StatusOK, `{"symbol":"BTCUSDT","price":"123456789.123456789"}`)

	require.NoError(t, err)
	assert.Equal(t, "123456789.123456789", priceQuote.Price.String())
}

func TestBinancePriceProxy_FailsWithoutAUsablePrice(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unknown pair", statusCode: http.StatusBadRequest, body: `{"code":-1121,"msg":"Invalid symbol."}`},
		{name: "malformed", statusCode: http.StatusOK, body: `{`},
		{name: "zero price", statusCode: http.StatusOK, body: `{"symbol":"BTCUSDT","price":"0"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := fetchFrom(t, testCase.statusCode, testCase.body)

			assert.Error(t, err)
		})
	}
}
