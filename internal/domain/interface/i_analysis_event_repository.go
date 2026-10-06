package interfaces

import (
	"context"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type IAnalysisEventRepository interface {
	Create(ctx context.Context, analysisEvent *entities.AnalysisEvent) error
	FindByID(ctx context.Context, analysisEventID uint) (*entities.AnalysisEvent, error)
	FindLatestReusable(ctx context.Context, symbol string, category string) (*entities.AnalysisEvent, error)
	Update(ctx context.Context, analysisEvent *entities.AnalysisEvent) error
	FailAllRunning(ctx context.Context, failureReason string, finishedAt time.Time) error
}
