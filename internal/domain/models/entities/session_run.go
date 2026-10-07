package entities

import "time"

type SessionRun struct {
	ID                   uint      `gorm:"primaryKey"`
	TradingDay           string    `gorm:"size:10;not null;uniqueIndex:idx_session_runs_day_session"`
	Session              string    `gorm:"size:16;not null;uniqueIndex:idx_session_runs_day_session"`
	Status               string    `gorm:"size:16;not null;index"`
	FailureReason        string    `gorm:"size:200"`
	SucceededSymbolCount int       `gorm:"not null"`
	FailedSymbolCount    int       `gorm:"not null"`
	StartedAt            time.Time `gorm:"not null"`
	FinishedAt           *time.Time
}
