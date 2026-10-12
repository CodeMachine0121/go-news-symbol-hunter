package vo

type BacktestPriceWindowsVo struct {
	Minute         PriceHistoryQueryVo
	HourlyFallback PriceHistoryQueryVo
	Daily          PriceHistoryQueryVo
}
