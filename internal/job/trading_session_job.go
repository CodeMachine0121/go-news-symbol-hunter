package job

import (
	"context"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/application"
)

type TradingSessionJob struct {
	sessionGradeApplication *application.SessionGradeApplication
	checkInterval           time.Duration
}

func NewTradingSessionJob(sessionGradeApplication *application.SessionGradeApplication, checkInterval time.Duration) *TradingSessionJob {
	return &TradingSessionJob{sessionGradeApplication: sessionGradeApplication, checkInterval: checkInterval}
}

// a non-positive check interval disables the job
func (tradingSessionJob *TradingSessionJob) Start(ctx context.Context) {
	if tradingSessionJob.checkInterval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(tradingSessionJob.checkInterval)
		defer ticker.Stop()
		for {
			// runs on this goroutine, so a long session drops ticks instead of overlapping the next check
			tradingSessionJob.sessionGradeApplication.RunDueTradingSession(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
