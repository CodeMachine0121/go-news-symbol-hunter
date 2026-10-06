package dto

import "time"

type AnalysisEventDto struct {
	AnalysisEventID uint               `json:"analysisEventId"`
	Symbol          string             `json:"symbol"`
	Category        string             `json:"category"`
	Status          string             `json:"status"`
	FailureReason   string             `json:"failureReason,omitempty"`
	StartedAt       time.Time          `json:"startedAt"`
	FinishedAt      *time.Time         `json:"finishedAt,omitempty"`
	Result          *AnalysisResultDto `json:"result,omitempty"`
}
