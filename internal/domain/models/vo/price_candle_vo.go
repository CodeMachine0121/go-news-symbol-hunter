package vo

import (
	"time"

	"github.com/shopspring/decimal"
)

type PriceCandleVo struct {
	OpenTime time.Time
	Close    decimal.Decimal
}
