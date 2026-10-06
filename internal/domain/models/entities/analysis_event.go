package entities

import "time"

type AnalysisEvent struct {
	ID            uint   `gorm:"primaryKey"`
	ApiKeyID      uint   `gorm:"not null;index"`
	Symbol        string `gorm:"size:32;not null;index:idx_analysis_events_lookup;uniqueIndex:idx_analysis_events_single_running,where:status = 'running'"`
	Category      string `gorm:"size:16;not null;index:idx_analysis_events_lookup;uniqueIndex:idx_analysis_events_single_running,where:status = 'running'"`
	Status        string `gorm:"size:16;not null;index:idx_analysis_events_lookup"`
	FailureReason string `gorm:"size:200"`
	Model         string `gorm:"size:100"`
	InputTokens   int64
	OutputTokens  int64
	StartedAt     time.Time `gorm:"not null"`
	FinishedAt    *time.Time
}
