package domains

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type TrackedSymbolGradeDomain struct {
	combinedGrade entities.CombinedGrade
	sessionGrades []entities.SessionGrade
}

func NewTrackedSymbolGradeDomain(combinedGrade entities.CombinedGrade, sessionGrades []entities.SessionGrade) TrackedSymbolGradeDomain {
	return TrackedSymbolGradeDomain{combinedGrade: combinedGrade, sessionGrades: sessionGrades}
}

func (trackedSymbolGradeDomain TrackedSymbolGradeDomain) ToDto() dto.TrackedSymbolGradeDto {
	sessionGradeDtos := make([]dto.SessionGradeDto, 0, len(trackedSymbolGradeDomain.sessionGrades))
	for _, sessionGrade := range trackedSymbolGradeDomain.sessionGrades {
		keyEvents := make([]dto.AnalysisKeyEventDto, 0, len(sessionGrade.KeyEvents))
		for _, keyEvent := range sessionGrade.KeyEvents {
			keyEvents = append(keyEvents, dto.AnalysisKeyEventDto{Title: keyEvent.Title, Link: keyEvent.Link, PublishedAt: keyEvent.PublishedAt})
		}
		evidence := make([]dto.AnalysisEvidenceDto, 0, len(sessionGrade.Evidence))
		for _, newsEvidence := range sessionGrade.Evidence {
			evidence = append(evidence, dto.AnalysisEvidenceDto{Title: newsEvidence.Title, Link: newsEvidence.Link, PublishedAt: newsEvidence.PublishedAt, ProviderName: newsEvidence.ProviderName})
		}
		sessionGradeDtos = append(sessionGradeDtos, dto.SessionGradeDto{
			Session:     sessionGrade.Session,
			Grade:       sessionGrade.Grade,
			Confidence:  sessionGrade.Confidence,
			Reason:      sessionGrade.Reason,
			KeyEvents:   keyEvents,
			RiskFactors: append([]string{}, sessionGrade.RiskFactors...),
			Evidence:    evidence,
			CreatedAt:   sessionGrade.CreatedAt,
		})
	}
	return dto.TrackedSymbolGradeDto{
		Symbol:        trackedSymbolGradeDomain.combinedGrade.Symbol,
		Category:      trackedSymbolGradeDomain.combinedGrade.Category,
		TradingDay:    trackedSymbolGradeDomain.combinedGrade.TradingDay,
		Grade:         trackedSymbolGradeDomain.combinedGrade.Grade,
		CombinedScore: trackedSymbolGradeDomain.combinedGrade.CombinedScore,
		Confidence:    trackedSymbolGradeDomain.combinedGrade.Confidence,
		UpdatedAt:     trackedSymbolGradeDomain.combinedGrade.UpdatedAt,
		SessionGrades: sessionGradeDtos,
	}
}
