package vo_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewPriceQuoteVo(t *testing.T) {
	pricedAt := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name          string
		price         string
		expectedError error
	}{
		{name: "positive price", price: "0.00001234"},
		{name: "zero price", price: "0", expectedError: vo.ErrPriceNotPositive},
		{name: "negative price", price: "-1", expectedError: vo.ErrPriceNotPositive},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			priceQuote, err := vo.NewPriceQuoteVo(decimal.RequireFromString(testCase.price), "USDT", pricedAt, "Binance")

			assert.ErrorIs(t, err, testCase.expectedError)
			if testCase.expectedError == nil {
				assert.Equal(t, vo.PriceQuoteVo{Price: decimal.RequireFromString(testCase.price), Currency: "USDT", PricedAt: pricedAt, Source: "Binance"}, priceQuote)
			}
		})
	}
}
