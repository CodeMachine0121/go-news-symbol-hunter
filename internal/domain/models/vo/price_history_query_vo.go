package vo

import "time"

const (
	PriceCandleIntervalMinute = "1m"
	PriceCandleIntervalHour   = "1h"
	PriceCandleIntervalDay    = "1d"
)

// Since and Until are both inclusive
type PriceHistoryQueryVo struct {
	Symbol   string
	Category string
	Interval string
	Since    time.Time
	Until    time.Time
}
