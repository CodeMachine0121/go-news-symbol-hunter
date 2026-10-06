package utilities_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/utilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaiwanExchangeQuoteParser_ParsesTradingDatesAsTaipeiMidnight(t *testing.T) {
	parsedDate, err := utilities.NewTaiwanExchangeQuoteParser().ParseTradingDate(" 1151006 ")

	require.NoError(t, err)
	assert.True(t, parsedDate.Equal(time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC)))
	_, offset := parsedDate.Zone()
	assert.Equal(t, 8*60*60, offset)
}

func TestTaiwanExchangeQuoteParser_RejectsMalformedDates(t *testing.T) {
	for _, rawDate := range []string{"", "115", "ABC1005", "1151399"} {
		_, err := utilities.NewTaiwanExchangeQuoteParser().ParseTradingDate(rawDate)

		assert.Error(t, err, rawDate)
	}
}

func TestTaiwanExchangeQuoteParser_ParsesClosingPrices(t *testing.T) {
	testCases := []struct {
		rawClosingPrice string
		expectedPrice   string
		expectsError    bool
	}{
		{rawClosingPrice: " 128.00 ", expectedPrice: "128"},
		{rawClosingPrice: "1,234.50", expectedPrice: "1234.5"},
		{rawClosingPrice: " ---", expectsError: true},
		{rawClosingPrice: "", expectsError: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.rawClosingPrice, func(t *testing.T) {
			closingPrice, err := utilities.NewTaiwanExchangeQuoteParser().ParseClosingPrice(testCase.rawClosingPrice)

			if testCase.expectsError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedPrice, closingPrice.String())
		})
	}
}
