package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AnalysisEventRepository struct {
	database *gorm.DB
}

func NewAnalysisEventRepository(database *gorm.DB) *AnalysisEventRepository {
	return &AnalysisEventRepository{database: database}
}

func (analysisEventRepository *AnalysisEventRepository) Create(ctx context.Context, analysisEvent *entities.AnalysisEvent) error {
	err := analysisEventRepository.database.WithContext(ctx).Create(analysisEvent).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return service.ErrAnalysisAlreadyRunning
	}
	return err
}

func (analysisEventRepository *AnalysisEventRepository) FindByID(ctx context.Context, analysisEventID uint) (*entities.AnalysisEvent, error) {
	var analysisEvent entities.AnalysisEvent
	err := analysisEventRepository.database.WithContext(ctx).Where(clause.Eq{Column: clause.Column{Name: "id"}, Value: analysisEventID}).First(&analysisEvent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &analysisEvent, nil
}

func (analysisEventRepository *AnalysisEventRepository) FindLatestReusable(ctx context.Context, symbol string, category string) (*entities.AnalysisEvent, error) {
	var analysisEvent entities.AnalysisEvent
	err := analysisEventRepository.database.WithContext(ctx).
		Where(clause.Eq{Column: clause.Column{Name: "symbol"}, Value: symbol}).
		Where(clause.Eq{Column: clause.Column{Name: "category"}, Value: category}).
		Where(clause.IN{Column: clause.Column{Name: "status"}, Values: []any{domains.AnalysisStatusRunning, domains.AnalysisStatusSucceeded}}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "started_at"}, Desc: true}).
		First(&analysisEvent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &analysisEvent, nil
}

func (analysisEventRepository *AnalysisEventRepository) Update(ctx context.Context, analysisEvent *entities.AnalysisEvent) error {
	return analysisEventRepository.database.WithContext(ctx).Save(analysisEvent).Error
}

func (analysisEventRepository *AnalysisEventRepository) FailAllRunning(ctx context.Context, failureReason string, finishedAt time.Time) error {
	return analysisEventRepository.database.WithContext(ctx).Model(&entities.AnalysisEvent{}).
		Where(clause.Eq{Column: clause.Column{Name: "status"}, Value: domains.AnalysisStatusRunning}).
		Updates(entities.AnalysisEvent{Status: domains.AnalysisStatusFailed, FailureReason: failureReason, FinishedAt: &finishedAt}).Error
}
