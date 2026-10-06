package dto

type StartSymbolAnalysisDto struct {
	ApiKeyID uint
	Symbol   string
	Category string
	// false when no analysis capacity is left: only an existing analysis may be returned
	AllowsNewAnalysis bool
}
