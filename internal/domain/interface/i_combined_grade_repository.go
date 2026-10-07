package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type ICombinedGradeRepository interface {
	// replaces the combined grade the symbol already has for the same trading day
	Save(ctx context.Context, combinedGrade *entities.CombinedGrade) error
	// empty symbol and category return every combined grade, newest trading day first
	FindAll(ctx context.Context, symbol string, category string) ([]entities.CombinedGrade, error)
	DeleteExceptTradingDay(ctx context.Context, tradingDay string) error
}
