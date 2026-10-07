package dto

import "time"

type TrackedSymbolGradeDto struct {
	Symbol        string            `json:"symbol"`
	Category      string            `json:"category"`
	TradingDay    string            `json:"tradingDay"`
	Grade         string            `json:"grade"`
	CombinedScore float64           `json:"combinedScore"`
	Confidence    int               `json:"confidence"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	SessionGrades []SessionGradeDto `json:"sessionGrades"`
}
