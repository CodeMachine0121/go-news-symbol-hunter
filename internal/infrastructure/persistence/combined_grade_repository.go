package persistence

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CombinedGradeRepository struct {
	database *gorm.DB
}

func NewCombinedGradeRepository(database *gorm.DB) *CombinedGradeRepository {
	return &CombinedGradeRepository{database: database}
}

func (combinedGradeRepository *CombinedGradeRepository) Save(ctx context.Context, combinedGrade *entities.CombinedGrade) error {
	return combinedGradeRepository.database.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}, {Name: "category"}, {Name: "trading_day"}},
		UpdateAll: true,
	}).Create(combinedGrade).Error
}

func (combinedGradeRepository *CombinedGradeRepository) FindAll(ctx context.Context, symbol string, category string) ([]entities.CombinedGrade, error) {
	combinedGrades := []entities.CombinedGrade{}
	query := combinedGradeRepository.database.WithContext(ctx)
	if symbol != "" {
		query = query.Where(clause.Eq{Column: clause.Column{Name: "symbol"}, Value: symbol}).Where(clause.Eq{Column: clause.Column{Name: "category"}, Value: category})
	}
	err := query.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "trading_day"}, Desc: true}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "symbol"}}).
		Find(&combinedGrades).Error
	return combinedGrades, err
}

func (combinedGradeRepository *CombinedGradeRepository) DeleteExceptTradingDay(ctx context.Context, tradingDay string) error {
	return combinedGradeRepository.database.WithContext(ctx).
		Where(clause.Neq{Column: clause.Column{Name: "trading_day"}, Value: tradingDay}).
		Delete(&entities.CombinedGrade{}).Error
}
