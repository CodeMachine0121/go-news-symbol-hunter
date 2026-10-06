package dto

type SymbolNewsDto struct {
	Symbol              string    `json:"symbol"`
	Category            string    `json:"category"`
	News                []NewsDto `json:"news"`
	FailedNewsProviders []string  `json:"failedNewsProviders"`
}
