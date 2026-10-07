package entities

import "time"

type CombinedGrade struct {
	ID            uint    `gorm:"primaryKey"`
	Symbol        string  `gorm:"size:32;not null;uniqueIndex:idx_combined_grades_symbol_day"`
	Category      string  `gorm:"size:16;not null;uniqueIndex:idx_combined_grades_symbol_day"`
	TradingDay    string  `gorm:"size:10;not null;uniqueIndex:idx_combined_grades_symbol_day;index"`
	Grade         string  `gorm:"size:16;not null"`
	CombinedScore float64 `gorm:"not null"`
	Confidence    int     `gorm:"not null"`
	UpdatedAt     time.Time
}
