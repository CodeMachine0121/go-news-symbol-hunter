package persistence

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TrackedSymbolRepository struct {
	database *gorm.DB
}

func NewTrackedSymbolRepository(database *gorm.DB) *TrackedSymbolRepository {
	return &TrackedSymbolRepository{database: database}
}

func (trackedSymbolRepository *TrackedSymbolRepository) FindTracking(ctx context.Context, category string) ([]entities.TrackedSymbol, error) {
	trackedSymbols := []entities.TrackedSymbol{}
	err := trackedSymbolRepository.database.WithContext(ctx).
		Where(clause.Eq{Column: clause.Column{Name: "category"}, Value: category}).
		Where(clause.Eq{Column: clause.Column{Name: "is_tracking"}, Value: true}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}}).
		Find(&trackedSymbols).Error
	return trackedSymbols, err
}
