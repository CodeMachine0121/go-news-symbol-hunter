package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

type PriceAtAnalysisDto struct {
	Price    decimal.Decimal `json:"price"`
	Currency string          `json:"currency"`
	PricedAt time.Time       `json:"pricedAt"`
	Source   string          `json:"source"`
}
