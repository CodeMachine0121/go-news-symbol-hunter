package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type ITrackedSymbolRepository interface {
	FindTracking(ctx context.Context, category string) ([]entities.TrackedSymbol, error)
}
