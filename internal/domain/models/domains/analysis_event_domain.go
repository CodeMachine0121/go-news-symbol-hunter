package domains

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	AnalysisStatusRunning   = "running"
	AnalysisStatusSucceeded = "succeeded"
	AnalysisStatusFailed    = "failed"
	AnalysisReuseWindow     = 6 * time.Hour
	AnalysisTimeout         = 10 * time.Minute
	AnalysisStaleAfter      = AnalysisTimeout + 5*time.Minute
)

type AnalysisEventDomain struct {
	analysisEvent entities.AnalysisEvent
}

func NewAnalysisEventDomain(analysisEvent entities.AnalysisEvent) AnalysisEventDomain {
	return AnalysisEventDomain{analysisEvent: analysisEvent}
}

func NewStartedAnalysisEventDomain(apiKeyID uint, symbol vo.SymbolVo, model string, startedAt time.Time) AnalysisEventDomain {
	return AnalysisEventDomain{analysisEvent: entities.AnalysisEvent{
		ApiKeyID:  apiKeyID,
		Symbol:    symbol.Value,
		Category:  symbol.Category.Value,
		Status:    AnalysisStatusRunning,
		Model:     model,
		StartedAt: startedAt,
	}}
}

func (analysisEventDomain AnalysisEventDomain) IsReusableAt(now time.Time) bool {
	switch analysisEventDomain.analysisEvent.Status {
	case AnalysisStatusRunning:
		return !analysisEventDomain.IsStaleAt(now)
	case AnalysisStatusSucceeded:
		finishedAt := analysisEventDomain.analysisEvent.FinishedAt
		return finishedAt != nil && !finishedAt.Before(now.Add(-AnalysisReuseWindow))
	default:
		return false
	}
}

func (analysisEventDomain AnalysisEventDomain) IsStaleAt(now time.Time) bool {
	return analysisEventDomain.analysisEvent.Status == AnalysisStatusRunning && analysisEventDomain.analysisEvent.StartedAt.Before(now.Add(-AnalysisStaleAfter))
}

func (analysisEventDomain AnalysisEventDomain) IsSucceeded() bool {
	return analysisEventDomain.analysisEvent.Status == AnalysisStatusSucceeded
}

func (analysisEventDomain *AnalysisEventDomain) Succeed(finishedAt time.Time, usage vo.AnalystUsageVo) {
	analysisEventDomain.finish(AnalysisStatusSucceeded, "", finishedAt, usage)
}

func (analysisEventDomain *AnalysisEventDomain) RecordAnsweringModel(modelName string) {
	if modelName != "" {
		analysisEventDomain.analysisEvent.Model = modelName
	}
}

func (analysisEventDomain *AnalysisEventDomain) Fail(failureReason string, finishedAt time.Time, usage vo.AnalystUsageVo) {
	analysisEventDomain.finish(AnalysisStatusFailed, failureReason, finishedAt, usage)
}

func (analysisEventDomain AnalysisEventDomain) ToDto(analysisResult *entities.AnalysisResult) dto.AnalysisEventDto {
	analysisEventDto := dto.AnalysisEventDto{
		AnalysisEventID: analysisEventDomain.analysisEvent.ID,
		Symbol:          analysisEventDomain.analysisEvent.Symbol,
		Category:        analysisEventDomain.analysisEvent.Category,
		Status:          analysisEventDomain.analysisEvent.Status,
		FailureReason:   analysisEventDomain.analysisEvent.FailureReason,
		StartedAt:       analysisEventDomain.analysisEvent.StartedAt,
		FinishedAt:      analysisEventDomain.analysisEvent.FinishedAt,
	}
	if analysisResult == nil {
		return analysisEventDto
	}
	keyEvents := make([]dto.AnalysisKeyEventDto, 0, len(analysisResult.KeyEvents))
	for _, keyEvent := range analysisResult.KeyEvents {
		keyEvents = append(keyEvents, dto.AnalysisKeyEventDto{Title: keyEvent.Title, Link: keyEvent.Link, PublishedAt: keyEvent.PublishedAt})
	}
	evidence := make([]dto.AnalysisEvidenceDto, 0, len(analysisResult.Evidence))
	for _, newsEvidence := range analysisResult.Evidence {
		evidence = append(evidence, dto.AnalysisEvidenceDto{Title: newsEvidence.Title, Link: newsEvidence.Link, PublishedAt: newsEvidence.PublishedAt, ProviderName: newsEvidence.ProviderName})
	}
	priceAtAnalysis := (*dto.PriceAtAnalysisDto)(nil)
	if analysisResult.Price.Valid && analysisResult.PricedAt != nil {
		priceAtAnalysis = &dto.PriceAtAnalysisDto{Price: analysisResult.Price.Decimal, Currency: analysisResult.PriceCurrency, PricedAt: *analysisResult.PricedAt, Source: analysisResult.PriceSource}
	}
	analysisEventDto.Result = &dto.AnalysisResultDto{
		AnalysisEventID: analysisResult.AnalysisEventID,
		Symbol:          analysisResult.Symbol,
		Category:        analysisResult.Category,
		Grade:           analysisResult.Grade,
		Confidence:      analysisResult.Confidence,
		TimeHorizon:     analysisResult.TimeHorizon,
		Reason:          analysisResult.Reason,
		KeyEvents:       keyEvents,
		RiskFactors:     append([]string{}, analysisResult.RiskFactors...),
		Evidence:        evidence,
		PriceAtAnalysis: priceAtAnalysis,
		CreatedAt:       analysisResult.CreatedAt,
	}
	return analysisEventDto
}

func (analysisEventDomain AnalysisEventDomain) ToEntity() entities.AnalysisEvent {
	return analysisEventDomain.analysisEvent
}

func (analysisEventDomain *AnalysisEventDomain) finish(status string, failureReason string, finishedAt time.Time, usage vo.AnalystUsageVo) {
	analysisEventDomain.analysisEvent.Status = status
	analysisEventDomain.analysisEvent.FailureReason = failureReason
	analysisEventDomain.analysisEvent.FinishedAt = &finishedAt
	analysisEventDomain.analysisEvent.InputTokens = usage.InputTokens
	analysisEventDomain.analysisEvent.OutputTokens = usage.OutputTokens
}
