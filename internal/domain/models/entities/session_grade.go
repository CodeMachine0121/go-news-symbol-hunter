package entities

import "time"

type SessionGrade struct {
	ID           uint               `gorm:"primaryKey"`
	Symbol       string             `gorm:"size:32;not null;uniqueIndex:idx_session_grades_symbol_session"`
	Category     string             `gorm:"size:16;not null;uniqueIndex:idx_session_grades_symbol_session"`
	TradingDay   string             `gorm:"size:10;not null;uniqueIndex:idx_session_grades_symbol_session;index"`
	Session      string             `gorm:"size:16;not null;uniqueIndex:idx_session_grades_symbol_session"`
	Grade        string             `gorm:"size:16;not null"`
	Confidence   int                `gorm:"not null"`
	Reason       string             `gorm:"type:text;not null"`
	KeyEvents    []AnalysisKeyEvent `gorm:"type:jsonb;serializer:json;not null"`
	RiskFactors  []string           `gorm:"type:jsonb;serializer:json;not null"`
	Evidence     []AnalysisEvidence `gorm:"type:jsonb;serializer:json;not null"`
	Model        string             `gorm:"size:100"`
	InputTokens  int64
	OutputTokens int64
	CreatedAt    time.Time
}
