package dto

import "time"

type NewsDto struct {
	Title        string    `json:"title"`
	Link         string    `json:"link"`
	PublishedAt  time.Time `json:"publishedAt"`
	ProviderName string    `json:"providerName"`
	Summary      string    `json:"summary"`
}
