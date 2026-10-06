package vo

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var ErrPriceNotPositive = errors.New("price must be positive")

type PriceQuoteVo struct {
	Price    decimal.Decimal
	Currency string
	PricedAt time.Time
	Source   string
}

func NewPriceQuoteVo(price decimal.Decimal, currency string, pricedAt time.Time, source string) (PriceQuoteVo, error) {
	if !price.IsPositive() {
		return PriceQuoteVo{}, ErrPriceNotPositive
	}
	return PriceQuoteVo{Price: price, Currency: currency, PricedAt: pricedAt, Source: source}, nil
}
