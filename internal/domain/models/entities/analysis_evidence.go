package entities

import "time"

type AnalysisEvidence struct {
	Title        string    `json:"title"`
	Link         string    `json:"link"`
	PublishedAt  time.Time `json:"publishedAt"`
	ProviderName string    `json:"providerName"`
}
