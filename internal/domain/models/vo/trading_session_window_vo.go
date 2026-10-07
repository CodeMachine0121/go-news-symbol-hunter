package vo

import "time"

// offsets are measured from midnight of the trading day; opening is inclusive, closing exclusive
type TradingSessionWindowVo struct {
	Session                         string
	OpensAt                         time.Duration
	ClosesAt                        time.Duration
	NewsSince                       time.Duration
	IsNewsSinceOnPreviousTradingDay bool
}
