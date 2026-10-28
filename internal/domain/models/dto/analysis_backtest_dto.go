package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

type AnalysisBacktestDto struct {
	GradeRecordID uint   `json:"gradeRecordId"`
	Source        string `json:"source"`
	// only manual records have an analysis event
	AnalysisEventID *uint     `json:"analysisEventId"`
	Symbol          string    `json:"symbol"`
	Category        string    `json:"category"`
	Grade           string    `json:"grade"`
	Confidence      int       `json:"confidence"`
	ConfidenceBand  string    `json:"confidenceBand"`
	TimeHorizon     string    `json:"timeHorizon"`
	CompletedAt     time.Time `json:"completedAt"`
	// only background records have a trading day
	TradingDay      *string          `json:"tradingDay"`
	EntryPrice      *decimal.Decimal `json:"entryPrice"`
	CheckpointDate  *string          `json:"checkpointDate"`
	CheckpointPrice *decimal.Decimal `json:"checkpointPrice"`
	PriceChangeRate *decimal.Decimal `json:"priceChangeRate"`
	Outcome         string           `json:"outcome"`
}
