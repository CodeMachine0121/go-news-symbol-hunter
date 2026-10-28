package vo

// a blank Source covers both manual and background grade records
type GradeBacktestFilterVo struct {
	CompletedPeriod AnalysisCompletedPeriodVo
	SymbolFilterVo
	Source string
}
