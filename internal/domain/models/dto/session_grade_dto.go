package dto

import "time"

type SessionGradeDto struct {
	Session     string                `json:"session"`
	Grade       string                `json:"grade"`
	Confidence  int                   `json:"confidence"`
	Reason      string                `json:"reason"`
	KeyEvents   []AnalysisKeyEventDto `json:"keyEvents"`
	RiskFactors []string              `json:"riskFactors"`
	Evidence    []AnalysisEvidenceDto `json:"evidence"`
	CreatedAt   time.Time             `json:"createdAt"`
}
