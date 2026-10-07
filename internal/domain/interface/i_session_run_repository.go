package interfaces

import (
	"context"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type ISessionRunRepository interface {
	// returns ErrSessionRunAlreadyExists when the trading day and session already ran
	Create(ctx context.Context, sessionRun *entities.SessionRun) error
	Update(ctx context.Context, sessionRun *entities.SessionRun) error
	FailAllRunning(ctx context.Context, failureReason string, finishedAt time.Time) error
}
