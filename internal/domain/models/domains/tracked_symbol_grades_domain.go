package domains

import (
	"cmp"
	"slices"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

type TrackedSymbolGradesDomain struct {
	// a symbol is tracking exactly when it has an entry, even an empty name
	symbolNames map[vo.SymbolVo]string
}

func NewTrackedSymbolGradesDomain(trackingSymbols []entities.TrackedSymbol) TrackedSymbolGradesDomain {
	symbolNames := map[vo.SymbolVo]string{}
	for _, trackingSymbol := range trackingSymbols {
		// administrators enter symbols by hand, so they are compared in the same normalized form grades are stored in
		symbol, err := vo.NewSymbolVo(trackingSymbol.Symbol, vo.MarketCategoryVo{Value: trackingSymbol.Category})
		if err == nil {
			// a duplicate row that normalizes to the same symbol must not blank out a registered name
			if symbolName := vo.NewSymbolNameVo(trackingSymbol.SymbolName).Value; symbolName != "" || symbolNames[symbol] == "" {
				symbolNames[symbol] = symbolName
			}
		}
	}
	return TrackedSymbolGradesDomain{symbolNames: symbolNames}
}

// each still-tracked symbol once in a stable order, narrowed to the filter when it names one; paused or removed symbols are left out
func (trackedSymbolGradesDomain TrackedSymbolGradesDomain) TrackingSymbols(symbolFilter vo.SymbolFilterVo) []vo.SymbolVo {
	trackingSymbols := []vo.SymbolVo{}
	for symbol := range trackedSymbolGradesDomain.symbolNames {
		if symbolFilter.Symbol == "" || (symbol.Value == symbolFilter.Symbol && symbol.Category.Value == symbolFilter.Category) {
			trackingSymbols = append(trackingSymbols, symbol)
		}
	}
	slices.SortFunc(trackingSymbols, func(left vo.SymbolVo, right vo.SymbolVo) int {
		return cmp.Or(cmp.Compare(left.Category.Value, right.Category.Value), cmp.Compare(left.Value, right.Value))
	})
	return trackingSymbols
}

// latestGradeRecords holds at most one background record for each of TrackingSymbols; the result lists the newest trading day first
func (trackedSymbolGradesDomain TrackedSymbolGradesDomain) ToDtos(latestGradeRecords []entities.GradeRecord, sessionGrades []entities.SessionGrade) []dto.TrackedSymbolGradeDto {
	trackedSymbolGrades := []dto.TrackedSymbolGradeDto{}
	for _, latestGradeRecord := range latestGradeRecords {
		symbol := vo.SymbolVo{Value: latestGradeRecord.Symbol, Category: vo.MarketCategoryVo{Value: latestGradeRecord.Category}}
		symbolName := trackedSymbolGradesDomain.symbolNames[symbol]
		sessionGradeDtos := []dto.SessionGradeDto{}
		for _, sessionGrade := range sessionGrades {
			if sessionGrade.Symbol != latestGradeRecord.Symbol || sessionGrade.Category != latestGradeRecord.Category || sessionGrade.TradingDay != latestGradeRecord.TradingDay {
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
		combinedScore := 0.0
		if latestGradeRecord.CombinedScore != nil {
			combinedScore = *latestGradeRecord.CombinedScore
		}
		trackedSymbolGrades = append(trackedSymbolGrades, dto.TrackedSymbolGradeDto{
			Symbol:        latestGradeRecord.Symbol,
			SymbolName:    symbolName,
			Category:      latestGradeRecord.Category,
			TradingDay:    latestGradeRecord.TradingDay,
			Grade:         latestGradeRecord.Grade,
			CombinedScore: combinedScore,
			Confidence:    latestGradeRecord.Confidence,
			UpdatedAt:     latestGradeRecord.CreatedAt,
			SessionGrades: sessionGradeDtos,
		})
	}
	slices.SortFunc(trackedSymbolGrades, func(left dto.TrackedSymbolGradeDto, right dto.TrackedSymbolGradeDto) int {
		return cmp.Or(cmp.Compare(right.TradingDay, left.TradingDay), cmp.Compare(left.Symbol, right.Symbol), cmp.Compare(left.Category, right.Category))
	})
	return trackedSymbolGrades
}
