package interfaces

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
)

type ISessionGradeRepository interface {
	// replaces the grade the symbol already has for the same trading day and session
	Save(ctx context.Context, sessionGrade *entities.SessionGrade) error
	FindByTradingDay(ctx context.Context, symbol string, category string, tradingDay string) ([]entities.SessionGrade, error)
	// every symbol's session grades on any of the trading days
	FindByTradingDays(ctx context.Context, tradingDays []string) ([]entities.SessionGrade, error)
	DeleteExceptTradingDay(ctx context.Context, tradingDay string) error
}
