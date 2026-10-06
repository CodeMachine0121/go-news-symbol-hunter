package vo

type AnalystTurnVo struct {
	Reply        string
	NewsSearches []AnalystNewsSearchVo
	Conclusion   *RawAnalysisConclusionVo
	IsRefused    bool
	Usage        AnalystUsageVo
}
