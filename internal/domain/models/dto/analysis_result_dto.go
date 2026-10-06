package dto

import "time"

type AnalysisResultDto struct {
	AnalysisEventID uint                  `json:"analysisEventId"`
	Symbol          string                `json:"symbol"`
	Category        string                `json:"category"`
	Grade           string                `json:"grade"`
	Confidence      int                   `json:"confidence"`
	TimeHorizon     string                `json:"timeHorizon"`
	Reason          string                `json:"reason"`
	KeyEvents       []AnalysisKeyEventDto `json:"keyEvents"`
	RiskFactors     []string              `json:"riskFactors"`
	Evidence        []AnalysisEvidenceDto `json:"evidence"`
	PriceAtAnalysis *PriceAtAnalysisDto   `json:"priceAtAnalysis"`
	CreatedAt       time.Time             `json:"createdAt"`
}

type AnalysisKeyEventDto struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"publishedAt"`
}

type AnalysisEvidenceDto struct {
	Title        string    `json:"title"`
	Link         string    `json:"link"`
	PublishedAt  time.Time `json:"publishedAt"`
	ProviderName string    `json:"providerName"`
}
