package entities

import "time"

type TrackedSymbol struct {
	ID         uint   `gorm:"primaryKey"`
	Symbol     string `gorm:"size:32;not null;uniqueIndex:idx_tracked_symbols_symbol_category"`
	Category   string `gorm:"size:16;not null;uniqueIndex:idx_tracked_symbols_symbol_category"`
	IsTracking bool   `gorm:"not null;default:true"`
	CreatedAt  time.Time
}
