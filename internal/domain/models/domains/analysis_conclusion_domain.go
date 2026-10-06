package domains

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	GradeStrongBullish = "strongBullish"
	GradeBullish       = "bullish"
	GradeNeutral       = "neutral"
	GradeBearish       = "bearish"
	GradeStrongBearish = "strongBearish"

	TimeHorizonShort = "short"
	TimeHorizonMid   = "mid"

	MinimumConfidence      = 0
	MaximumConfidence      = 100
	MaximumKeyEventCount   = 5
	MaximumRiskFactorCount = 5
)

var (
	ErrAnalysisIncomplete = errors.New("AI 未提供完整分析")
	validGrades           = []string{GradeStrongBullish, GradeBullish, GradeNeutral, GradeBearish, GradeStrongBearish}
	validTimeHorizons     = []string{TimeHorizonShort, TimeHorizonMid}
)

type AnalysisConclusionDomain struct {
	grade       string
	confidence  int
	timeHorizon string
	reason      string
	keyEvents   []entities.AnalysisKeyEvent
	riskFactors []string
	evidence    []entities.AnalysisEvidence
}

func NewAnalysisConclusionDomain(rawConclusion vo.RawAnalysisConclusionVo, analysisEvidence *AnalysisEvidenceDomain) (AnalysisConclusionDomain, error) {
	reason := strings.TrimSpace(rawConclusion.Reason)
	if reason == "" {
		return AnalysisConclusionDomain{}, ErrAnalysisIncomplete
	}
	grade := GradeNeutral
	if slices.Contains(validGrades, rawConclusion.Grade) {
		grade = rawConclusion.Grade
	}
	timeHorizon := TimeHorizonShort
	if slices.Contains(validTimeHorizons, rawConclusion.TimeHorizon) {
		timeHorizon = rawConclusion.TimeHorizon
	}
	confidence := min(max(rawConclusion.Confidence, MinimumConfidence), MaximumConfidence)
	if analysisEvidence.IsEmpty() {
		grade = GradeNeutral
		confidence = MinimumConfidence
	}
	keyEvents := []entities.AnalysisKeyEvent{}
	for _, keyEventLink := range rawConclusion.KeyEventLinks {
		evidence, cited := analysisEvidence.FindByLink(strings.TrimSpace(keyEventLink))
		if cited && len(keyEvents) < MaximumKeyEventCount {
			keyEvents = append(keyEvents, entities.AnalysisKeyEvent{Title: evidence.Title, Link: evidence.Link, PublishedAt: evidence.PublishedAt})
		}
	}
	riskFactors := []string{}
	for _, rawRiskFactor := range rawConclusion.RiskFactors {
		riskFactor := strings.TrimSpace(rawRiskFactor)
		if riskFactor != "" && len(riskFactors) < MaximumRiskFactorCount {
			riskFactors = append(riskFactors, riskFactor)
		}
	}
	return AnalysisConclusionDomain{
		grade:       grade,
		confidence:  confidence,
		timeHorizon: timeHorizon,
		reason:      reason,
		keyEvents:   keyEvents,
		riskFactors: riskFactors,
		evidence:    analysisEvidence.ToEntities(),
	}, nil
}

func (analysisConclusionDomain AnalysisConclusionDomain) ToResultEntity(analysisEvent entities.AnalysisEvent, createdAt time.Time) entities.AnalysisResult {
	return entities.AnalysisResult{
		AnalysisEventID: analysisEvent.ID,
		Symbol:          analysisEvent.Symbol,
		Category:        analysisEvent.Category,
		Grade:           analysisConclusionDomain.grade,
		Confidence:      analysisConclusionDomain.confidence,
		TimeHorizon:     analysisConclusionDomain.timeHorizon,
		Reason:          analysisConclusionDomain.reason,
		KeyEvents:       analysisConclusionDomain.keyEvents,
		RiskFactors:     analysisConclusionDomain.riskFactors,
		Evidence:        analysisConclusionDomain.evidence,
		CreatedAt:       createdAt,
	}
}
