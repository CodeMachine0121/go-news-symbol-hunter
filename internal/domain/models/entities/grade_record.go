package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

// one graded conclusion, whether a user started the analysis (manual) or a tracked symbol's combined
// grade was recalculated (background); every record is kept, none is ever replaced
type GradeRecord struct {
	ID     uint   `gorm:"primaryKey"`
	Source string `gorm:"size:16;not null;index:idx_grade_records_source_symbol_day"`
	// only manual records belong to an analysis event; background ones leave it empty
	AnalysisEventID *uint  `gorm:"uniqueIndex"`
	Symbol          string `gorm:"size:32;not null;index:idx_grade_records_source_symbol_day"`
	Category        string `gorm:"size:16;not null;index:idx_grade_records_source_symbol_day"`
	SymbolName      string `gorm:"size:100;not null;default:''"`
	// only background records have a trading day; manual ones leave it blank
	TradingDay  string `gorm:"size:10;not null;default:'';index:idx_grade_records_source_symbol_day"`
	Grade       string `gorm:"size:16;not null"`
	Confidence  int    `gorm:"not null"`
	TimeHorizon string `gorm:"size:16;not null"`
	// only background records have a combined score
	CombinedScore *float64
	Reason        string              `gorm:"type:text;not null;default:''"`
	KeyEvents     []AnalysisKeyEvent  `gorm:"type:jsonb;serializer:json;not null"`
	RiskFactors   []string            `gorm:"type:jsonb;serializer:json;not null"`
	Evidence      []AnalysisEvidence  `gorm:"type:jsonb;serializer:json;not null"`
	Price         decimal.NullDecimal `gorm:"type:numeric"`
	PriceCurrency string              `gorm:"size:8"`
	PricedAt      *time.Time
	PriceSource   string    `gorm:"size:32"`
	CreatedAt     time.Time `gorm:"index:idx_grade_records_source_symbol_day"`
}
