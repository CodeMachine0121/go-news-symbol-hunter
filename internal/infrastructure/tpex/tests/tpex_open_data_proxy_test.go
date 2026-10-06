package tpex_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/httpfetch"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/infrastructure/tpex"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dailyClosings = `[
{"Date":"1151006","SecuritiesCompanyCode":"6182","CompanyName":"合晶","Close":"128.00"},
{"Date":"1151006","SecuritiesCompanyCode":" 8299 ","CompanyName":" 群聯 ","Close":"1,234.50"},
{"Date":"1151006","SecuritiesCompanyCode":"5483","CompanyName":"中美晶","Close":" ---"},
{"Date":"1151006","SecuritiesCompanyCode":"","CompanyName":"沒有代號","Close":"10"},
{"Date":"1151006","SecuritiesCompanyCode":"9998","CompanyName":"","Close":"10"},
{"Date":"ABC","SecuritiesCompanyCode":"9997","CompanyName":"壞日期","Close":"10"}
]`

var lookedUpAt = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func startServer(t *testing.T, statusCode int, body string) (*tpex.TpexOpenDataProxy, *atomic.Int32) {
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Maybe()
	return tpex.NewTpexOpenDataProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL), requestCount
}

func TestTpexOpenDataProxy_FindsOtcCompanyShortNames(t *testing.T) {
	tpexOpenDataProxy, requestCount := startServer(t, http.StatusOK, dailyClosings)

	shortName, found, err := tpexOpenDataProxy.FindCompanyShortName(context.Background(), "6182")
	trimmedShortName, trimmedFound, _ := tpexOpenDataProxy.FindCompanyShortName(context.Background(), "8299")
	_, unlistedFound, _ := tpexOpenDataProxy.FindCompanyShortName(context.Background(), "2330")
	_, unnamedFound, _ := tpexOpenDataProxy.FindCompanyShortName(context.Background(), "9998")

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "合晶", shortName)
	assert.True(t, trimmedFound)
	assert.Equal(t, "群聯", trimmedShortName)
	assert.False(t, unlistedFound)
	assert.False(t, unnamedFound)
	assert.Equal(t, int32(1), requestCount.Load())
}

func TestTpexOpenDataProxy_QuotesTheLatestClosingPrice(t *testing.T) {
	tpexOpenDataProxy, _ := startServer(t, http.StatusOK, dailyClosings)

	priceQuote, err := tpexOpenDataProxy.FetchPrice(context.Background(), "6182")
	thousandsQuote, thousandsError := tpexOpenDataProxy.FetchPrice(context.Background(), "8299")

	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("128.00").Equal(priceQuote.Price))
	assert.Equal(t, "TWD", priceQuote.Currency)
	assert.Equal(t, "櫃買中心", priceQuote.Source)
	assert.True(t, priceQuote.PricedAt.Equal(time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC)))
	require.NoError(t, thousandsError)
	assert.Equal(t, "1234.5", thousandsQuote.Price.String())
}

func TestTpexOpenDataProxy_HasNoPriceWithoutAUsableClose(t *testing.T) {
	tpexOpenDataProxy, _ := startServer(t, http.StatusOK, dailyClosings)

	for _, stockCode := range []string{"5483", "2330", "9997"} {
		_, err := tpexOpenDataProxy.FetchPrice(context.Background(), stockCode)

		assert.Error(t, err, stockCode)
	}
}

func TestTpexOpenDataProxy_ReportsUnavailableOrEmptyData(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unavailable", statusCode: http.StatusBadGateway},
		{name: "malformed", statusCode: http.StatusOK, body: `{`},
		{name: "empty", statusCode: http.StatusOK, body: `[]`},
		{name: "no usable rows", statusCode: http.StatusOK, body: `[{"SecuritiesCompanyCode":"","CompanyName":"x"}]`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tpexOpenDataProxy, _ := startServer(t, testCase.statusCode, testCase.body)

			_, found, lookupError := tpexOpenDataProxy.FindCompanyShortName(context.Background(), "6182")
			_, priceError := tpexOpenDataProxy.FetchPrice(context.Background(), "6182")

			assert.Error(t, lookupError)
			assert.False(t, found)
			assert.Error(t, priceError)
		})
	}
}

func TestTpexOpenDataProxy_ReusesTheDailyClosingsForAnHour(t *testing.T) {
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		_, _ = writer.Write([]byte(dailyClosings))
	}))
	defer server.Close()
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Times(2)
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(time.Hour - time.Second)).Once()
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(time.Hour))
	tpexOpenDataProxy := tpex.NewTpexOpenDataProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)

	_, _, _ = tpexOpenDataProxy.FindCompanyShortName(context.Background(), "6182")
	_, _ = tpexOpenDataProxy.FetchPrice(context.Background(), "6182")
	assert.Equal(t, int32(1), requestCount.Load())
	_, _, _ = tpexOpenDataProxy.FindCompanyShortName(context.Background(), "6182")

	assert.Equal(t, int32(2), requestCount.Load())
}

func TestTpexOpenDataProxy_KeepsTheLastGoodDataWhenARefreshFails(t *testing.T) {
	requestCount := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if requestCount.Add(1) > 1 {
			writer.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = writer.Write([]byte(dailyClosings))
	}))
	defer server.Close()
	clockProxy := mocks.NewMockIClockProxy(t)
	clockProxy.EXPECT().Now().Return(lookedUpAt).Times(2)
	clockProxy.EXPECT().Now().Return(lookedUpAt.Add(2 * time.Hour))
	tpexOpenDataProxy := tpex.NewTpexOpenDataProxy(httpfetch.NewHttpBodyReader(server.Client()), clockProxy, server.URL)
	_, _, _ = tpexOpenDataProxy.FindCompanyShortName(context.Background(), "6182")

	shortName, found, lookupError := tpexOpenDataProxy.FindCompanyShortName(context.Background(), "6182")
	priceQuote, priceError := tpexOpenDataProxy.FetchPrice(context.Background(), "6182")

	require.NoError(t, lookupError)
	assert.True(t, found)
	assert.Equal(t, "合晶", shortName)
	require.NoError(t, priceError)
	assert.True(t, decimal.RequireFromString("128.00").Equal(priceQuote.Price))
	assert.GreaterOrEqual(t, requestCount.Load(), int32(2))
}
