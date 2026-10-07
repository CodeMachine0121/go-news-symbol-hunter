package domains

import (
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
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
	return SessionRunDomain{sessionRun: entities.SessionRun{TradingDay: tradingDay, Session: session, Status: SessionRunStatusRunning, StartedAt: startedAt}}
}

func (sessionRunDomain *SessionRunDomain) RecordSymbolOutcome(isSucceeded bool) {
	if isSucceeded {
		sessionRunDomain.sessionRun.SucceededSymbolCount++
		return
	}
	sessionRunDomain.sessionRun.FailedSymbolCount++
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
