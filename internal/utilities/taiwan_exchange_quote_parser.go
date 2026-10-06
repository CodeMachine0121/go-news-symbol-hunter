package utilities

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const republicOfChinaYearOffset = 1911

var taipeiLocation = time.FixedZone("Asia/Taipei", 8*60*60)

type TaiwanExchangeQuoteParser struct{}

func NewTaiwanExchangeQuoteParser() *TaiwanExchangeQuoteParser {
	return &TaiwanExchangeQuoteParser{}
}

// exchanges print thousands separators and use placeholders such as "---" when a stock did not trade
func (taiwanExchangeQuoteParser *TaiwanExchangeQuoteParser) ParseClosingPrice(rawClosingPrice string) (decimal.Decimal, error) {
	closingPrice, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(rawClosingPrice), ",", ""))
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("no closing price in %q", rawClosingPrice)
	}
	return closingPrice, nil
}

// Taiwan exchanges publish dates in the Republic of China calendar, e.g. 1151005 is 2026-10-05
func (taiwanExchangeQuoteParser *TaiwanExchangeQuoteParser) ParseTradingDate(rawDate string) (time.Time, error) {
	trimmedDate := strings.TrimSpace(rawDate)
	if len(trimmedDate) < 7 {
		return time.Time{}, fmt.Errorf("unexpected Republic of China date %q", rawDate)
	}
	republicOfChinaYear, err := strconv.Atoi(trimmedDate[:len(trimmedDate)-4])
	if err != nil {
		return time.Time{}, fmt.Errorf("unexpected Republic of China date %q: %w", rawDate, err)
	}
	return time.ParseInLocation("20060102", strconv.Itoa(republicOfChinaYear+republicOfChinaYearOffset)+trimmedDate[len(trimmedDate)-4:], taipeiLocation)
}
