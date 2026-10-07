package domains

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const (
	SessionRunStatusRunning   = "running"
	SessionRunStatusSucceeded = "succeeded"
	SessionRunStatusFailed    = "failed"
)

type SessionRunDomain struct {
	sessionRun entities.SessionRun
}

func NewSessionRunDomain(sessionRun entities.SessionRun) SessionRunDomain {
	return SessionRunDomain{sessionRun: sessionRun}
}

func NewStartedSessionRunDomain(tradingDay string, session string, startedAt time.Time) SessionRunDomain {
	return SessionRunDomain{sessionRun: entities.SessionRun{TradingDay: tradingDay, Session: session, Status: SessionRunStatusRunning, FailedSymbols: []entities.SessionRunFailedSymbol{}, StartedAt: startedAt}}
}

func (sessionRunDomain *SessionRunDomain) RecordSymbolOutcome(symbolGradingOutcome vo.SymbolGradingOutcomeVo) {
	sessionRunDomain.sessionRun.InputTokens += symbolGradingOutcome.Usage.InputTokens
	sessionRunDomain.sessionRun.OutputTokens += symbolGradingOutcome.Usage.OutputTokens
	if symbolGradingOutcome.FailureReason == "" {
		sessionRunDomain.sessionRun.SucceededSymbolCount++
		return
	}
	sessionRunDomain.sessionRun.FailedSymbolCount++
	sessionRunDomain.sessionRun.FailedSymbols = append(sessionRunDomain.sessionRun.FailedSymbols, entities.SessionRunFailedSymbol{
		Symbol:        symbolGradingOutcome.Symbol,
		Category:      symbolGradingOutcome.Category,
		FailureReason: symbolGradingOutcome.FailureReason,
	})
}

func (sessionRunDomain *SessionRunDomain) Succeed(finishedAt time.Time) {
	sessionRunDomain.sessionRun.Status = SessionRunStatusSucceeded
	sessionRunDomain.sessionRun.FinishedAt = &finishedAt
}

func (sessionRunDomain *SessionRunDomain) Fail(failureReason string, finishedAt time.Time) {
	sessionRunDomain.sessionRun.Status = SessionRunStatusFailed
	sessionRunDomain.sessionRun.FailureReason = failureReason
	sessionRunDomain.sessionRun.FinishedAt = &finishedAt
}

func (sessionRunDomain SessionRunDomain) ToEntity() entities.SessionRun {
	return sessionRunDomain.sessionRun
}
