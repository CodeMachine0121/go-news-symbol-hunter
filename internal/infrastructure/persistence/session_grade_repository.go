package persistence

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SessionGradeRepository struct {
	database *gorm.DB
}

func NewSessionGradeRepository(database *gorm.DB) *SessionGradeRepository {
	return &SessionGradeRepository{database: database}
}

func (sessionGradeRepository *SessionGradeRepository) Save(ctx context.Context, sessionGrade *entities.SessionGrade) error {
	return sessionGradeRepository.database.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}, {Name: "category"}, {Name: "trading_day"}, {Name: "session"}},
		UpdateAll: true,
	}).Create(sessionGrade).Error
}

func (sessionGradeRepository *SessionGradeRepository) FindByTradingDay(ctx context.Context, symbol string, category string, tradingDay string) ([]entities.SessionGrade, error) {
	sessionGrades := []entities.SessionGrade{}
	err := sessionGradeRepository.database.WithContext(ctx).
		Where(clause.Eq{Column: clause.Column{Name: "symbol"}, Value: symbol}).
		Where(clause.Eq{Column: clause.Column{Name: "category"}, Value: category}).
		Where(clause.Eq{Column: clause.Column{Name: "trading_day"}, Value: tradingDay}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}}).
		Find(&sessionGrades).Error
	return sessionGrades, err
}

func (sessionGradeRepository *SessionGradeRepository) DeleteExceptTradingDay(ctx context.Context, tradingDay string) error {
	return sessionGradeRepository.database.WithContext(ctx).
		Where(clause.Neq{Column: clause.Column{Name: "trading_day"}, Value: tradingDay}).
		Delete(&entities.SessionGrade{}).Error
}
