package domains

import (
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type TrackedSymbolGradesDomain struct {
	trackingSymbols []entities.TrackedSymbol
	combinedGrades  []entities.CombinedGrade
	sessionGrades   []entities.SessionGrade
}

// combinedGrades must be ordered newest trading day first
func NewTrackedSymbolGradesDomain(trackingSymbols []entities.TrackedSymbol, combinedGrades []entities.CombinedGrade, sessionGrades []entities.SessionGrade) TrackedSymbolGradesDomain {
	return TrackedSymbolGradesDomain{trackingSymbols: trackingSymbols, combinedGrades: combinedGrades, sessionGrades: sessionGrades}
}

// each still-tracked symbol appears once, with its newest trading day; paused or removed symbols are left out
func (trackedSymbolGradesDomain TrackedSymbolGradesDomain) ToDtos() []dto.TrackedSymbolGradeDto {
	isTracking := map[vo.SymbolVo]bool{}
	for _, trackingSymbol := range trackedSymbolGradesDomain.trackingSymbols {
		// administrators enter symbols by hand, so they are compared in the same normalized form grades are stored in
		symbol, err := vo.NewSymbolVo(trackingSymbol.Symbol, vo.MarketCategoryVo{Value: trackingSymbol.Category})
		if err == nil {
			isTracking[symbol] = true
		}
	}
	isListed := map[vo.SymbolVo]bool{}
	trackedSymbolGrades := []dto.TrackedSymbolGradeDto{}
	for _, combinedGrade := range trackedSymbolGradesDomain.combinedGrades {
		symbol := vo.SymbolVo{Value: combinedGrade.Symbol, Category: vo.MarketCategoryVo{Value: combinedGrade.Category}}
		if !isTracking[symbol] || isListed[symbol] {
			continue
		}
		isListed[symbol] = true
		sessionGradeDtos := []dto.SessionGradeDto{}
		for _, sessionGrade := range trackedSymbolGradesDomain.sessionGrades {
			if sessionGrade.Symbol != combinedGrade.Symbol || sessionGrade.Category != combinedGrade.Category || sessionGrade.TradingDay != combinedGrade.TradingDay {
				continue
			}
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
		trackedSymbolGrades = append(trackedSymbolGrades, dto.TrackedSymbolGradeDto{
			Symbol:        combinedGrade.Symbol,
			Category:      combinedGrade.Category,
			TradingDay:    combinedGrade.TradingDay,
			Grade:         combinedGrade.Grade,
			CombinedScore: combinedGrade.CombinedScore,
			Confidence:    combinedGrade.Confidence,
			UpdatedAt:     combinedGrade.UpdatedAt,
			SessionGrades: sessionGradeDtos,
		})
	}
	return trackedSymbolGrades
}
