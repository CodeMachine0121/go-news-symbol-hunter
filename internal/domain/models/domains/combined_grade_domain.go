package domains

import (
	"math"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const combinedScorePrecision = 10000

var gradeScores = map[string]float64{
	GradeStrongBullish: 2,
	GradeBullish:       1,
	GradeNeutral:       0,
	GradeBearish:       -1,
	GradeStrongBearish: -2,
}

type CombinedGradeDomain struct {
	combinedScore float64
	grade         string
	confidence    int
}

func NewCombinedGradeDomain(sessionGrades []entities.SessionGrade, sessionWeights vo.SessionWeightsVo) CombinedGradeDomain {
	sessionWeightsBySession := map[string]float64{
		TradingSessionPreMarket:   sessionWeights.PreMarket,
		TradingSessionIntraday:    sessionWeights.Intraday,
		TradingSessionAfterMarket: sessionWeights.AfterMarket,
	}
	weightedScore := 0.0
	weightedConfidence := 0.0
	totalWeight := 0.0
	for _, sessionGrade := range sessionGrades {
		sessionWeight := sessionWeightsBySession[sessionGrade.Session]
		weightedScore += gradeScores[sessionGrade.Grade] * sessionWeight
		weightedConfidence += float64(sessionGrade.Confidence) * sessionWeight
		totalWeight += sessionWeight
	}
	if totalWeight == 0 {
		return CombinedGradeDomain{grade: GradeNeutral, confidence: MinimumConfidence}
	}
	// rounded so floating-point noise such as 0.49999999 cannot cross a grade threshold
	combinedScore := math.Round(weightedScore/totalWeight*combinedScorePrecision) / combinedScorePrecision
	grade := GradeStrongBearish
	switch {
	case combinedScore >= 1.5:
		grade = GradeStrongBullish
	case combinedScore >= 0.5:
		grade = GradeBullish
	case combinedScore > -0.5:
		grade = GradeNeutral
	case combinedScore > -1.5:
		grade = GradeBearish
	}
	return CombinedGradeDomain{combinedScore: combinedScore, grade: grade, confidence: int(math.Round(weightedConfidence / totalWeight))}
}

func (combinedGradeDomain CombinedGradeDomain) ToEntity(symbol vo.SymbolVo, tradingDay string, updatedAt time.Time) entities.CombinedGrade {
	return entities.CombinedGrade{
		Symbol:        symbol.Value,
		Category:      symbol.Category.Value,
		TradingDay:    tradingDay,
		Grade:         combinedGradeDomain.grade,
		CombinedScore: combinedGradeDomain.combinedScore,
		Confidence:    combinedGradeDomain.confidence,
		UpdatedAt:     updatedAt,
	}
}
