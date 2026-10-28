package entities

import "time"

type AnalysisKeyEvent struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"publishedAt"`
}
