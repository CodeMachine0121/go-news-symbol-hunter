package dto

import "github.com/shopspring/decimal"

type GradeBacktestDto struct {
	RandomBaseline           float64               `json:"randomBaseline"`
	Overall                  HitRateDto            `json:"overall"`
	ByCategory               map[string]HitRateDto `json:"byCategory"`
	ByGrade                  map[string]HitRateDto `json:"byGrade"`
	ByConfidenceBand         map[string]HitRateDto `json:"byConfidenceBand"`
	BySource                 map[string]HitRateDto `json:"bySource"`
	NeutralAverageChangeRate *decimal.Decimal      `json:"neutralAverageChangeRate"`
	OutcomeCounts            map[string]int        `json:"outcomeCounts"`
	Analyses                 []AnalysisBacktestDto `json:"analyses"`
}

// a nil HitRate means there is nothing to judge yet, which is not the same as a 0% hit rate
type HitRateDto struct {
	HitCount  int      `json:"hitCount"`
	MissCount int      `json:"missCount"`
	HitRate   *float64 `json:"hitRate"`
}
