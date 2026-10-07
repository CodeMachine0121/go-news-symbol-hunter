package vo

// an empty FailureReason means the symbol was graded
type SymbolGradingOutcomeVo struct {
	Symbol        string
	Category      string
	FailureReason string
	Usage         AnalystUsageVo
}
