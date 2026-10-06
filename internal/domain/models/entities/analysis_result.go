package entities

import "time"

type AnalysisResult struct {
	ID              uint               `gorm:"primaryKey"`
	AnalysisEventID uint               `gorm:"not null;uniqueIndex"`
	Symbol          string             `gorm:"size:32;not null"`
	Category        string             `gorm:"size:16;not null"`
	Grade           string             `gorm:"size:16;not null"`
	Confidence      int                `gorm:"not null"`
	TimeHorizon     string             `gorm:"size:16;not null"`
	Reason          string             `gorm:"type:text;not null"`
	KeyEvents       []AnalysisKeyEvent `gorm:"type:jsonb;serializer:json;not null"`
	RiskFactors     []string           `gorm:"type:jsonb;serializer:json;not null"`
	Evidence        []AnalysisEvidence `gorm:"type:jsonb;serializer:json;not null"`
	CreatedAt       time.Time
}

type AnalysisKeyEvent struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"publishedAt"`
}

type AnalysisEvidence struct {
	Title        string    `json:"title"`
	Link         string    `json:"link"`
	PublishedAt  time.Time `json:"publishedAt"`
	ProviderName string    `json:"providerName"`
}
