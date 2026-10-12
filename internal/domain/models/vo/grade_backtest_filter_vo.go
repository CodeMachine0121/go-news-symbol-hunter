package vo

// blank Symbol and Category cover every symbol
type GradeBacktestFilterVo struct {
	CompletedPeriod AnalysisCompletedPeriodVo
	Symbol          string
	Category        string
}
