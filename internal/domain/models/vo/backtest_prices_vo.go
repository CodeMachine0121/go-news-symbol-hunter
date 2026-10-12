package vo

// candles are earliest first
type BacktestPricesVo struct {
	MinuteCandles []PriceCandleVo
	HourlyCandles []PriceCandleVo
	DailyCandles  []PriceCandleVo
}
