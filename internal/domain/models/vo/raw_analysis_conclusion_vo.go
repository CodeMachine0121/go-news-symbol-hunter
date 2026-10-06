package vo

type RawAnalysisConclusionVo struct {
	Grade         string
	Confidence    int
	TimeHorizon   string
	Reason        string
	KeyEventLinks []string
	RiskFactors   []string
}
