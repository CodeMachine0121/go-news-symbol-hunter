package dto

type StartedSymbolAnalysisDto struct {
	AnalysisEvent AnalysisEventDto
	IsNew         bool
	SearchKeyword string
}
