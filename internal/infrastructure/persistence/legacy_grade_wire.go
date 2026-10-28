package persistence

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// the rows of the analysis_results table that grade records replaced; read once to move them over
type legacyAnalysisResult struct {
	ID              uint
	AnalysisEventID uint
	Symbol          string
	Category        string
	SymbolName      string
	Grade           string
	Confidence      int
	TimeHorizon     string
	Reason          string
	KeyEvents       []entities.AnalysisKeyEvent `gorm:"serializer:json"`
	RiskFactors     []string                    `gorm:"serializer:json"`
	Evidence        []entities.AnalysisEvidence `gorm:"serializer:json"`
	Price           decimal.NullDecimal
	PriceCurrency   string
	PricedAt        *time.Time
	PriceSource     string
	CreatedAt       time.Time
}

func (legacyAnalysisResult) TableName() string {
	return "analysis_results"
}

func (legacyAnalysisResult legacyAnalysisResult) toGradeRecord() entities.GradeRecord {
	return entities.GradeRecord{
		Source:          vo.GradeSourceManual,
		AnalysisEventID: &legacyAnalysisResult.AnalysisEventID,
		Symbol:          legacyAnalysisResult.Symbol,
		Category:        legacyAnalysisResult.Category,
		SymbolName:      legacyAnalysisResult.SymbolName,
		Grade:           legacyAnalysisResult.Grade,
		Confidence:      legacyAnalysisResult.Confidence,
		TimeHorizon:     legacyAnalysisResult.TimeHorizon,
		Reason:          legacyAnalysisResult.Reason,
		KeyEvents:       legacyAnalysisResult.KeyEvents,
		RiskFactors:     legacyAnalysisResult.RiskFactors,
		Evidence:        legacyAnalysisResult.Evidence,
		Price:           legacyAnalysisResult.Price,
		PriceCurrency:   legacyAnalysisResult.PriceCurrency,
		PricedAt:        legacyAnalysisResult.PricedAt,
		PriceSource:     legacyAnalysisResult.PriceSource,
		CreatedAt:       legacyAnalysisResult.CreatedAt,
	}
}

// the rows of the combined_grades table that background grade records replaced; read once to move them over
type legacyCombinedGrade struct {
	ID            uint
	Symbol        string
	Category      string
	TradingDay    string
	Grade         string
	CombinedScore float64
	Confidence    int
	UpdatedAt     time.Time
}

func (legacyCombinedGrade) TableName() string {
	return "combined_grades"
}

// a combined grade was last recalculated at its update time, which is when its background record would have been made
func (legacyCombinedGrade legacyCombinedGrade) toGradeRecord() entities.GradeRecord {
	return entities.GradeRecord{
		Source:        vo.GradeSourceBackground,
		Symbol:        legacyCombinedGrade.Symbol,
		Category:      legacyCombinedGrade.Category,
		TradingDay:    legacyCombinedGrade.TradingDay,
		Grade:         legacyCombinedGrade.Grade,
		Confidence:    legacyCombinedGrade.Confidence,
		TimeHorizon:   domains.TimeHorizonShort,
		CombinedScore: &legacyCombinedGrade.CombinedScore,
		KeyEvents:     []entities.AnalysisKeyEvent{},
		RiskFactors:   []string{},
		Evidence:      []entities.AnalysisEvidence{},
		CreatedAt:     legacyCombinedGrade.UpdatedAt,
	}
}
