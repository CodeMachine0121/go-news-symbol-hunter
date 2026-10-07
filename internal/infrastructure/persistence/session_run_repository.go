package persistence

import (
	"context"
	"time"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SessionRunRepository struct {
	database *gorm.DB
}

func NewSessionRunRepository(database *gorm.DB) *SessionRunRepository {
	return &SessionRunRepository{database: database}
}

func (sessionRunRepository *SessionRunRepository) Create(ctx context.Context, sessionRun *entities.SessionRun) error {
	// checked every minute inside a window, so an existing run is skipped quietly instead of failing the insert
	result := sessionRunRepository.database.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(sessionRun)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return service.ErrSessionRunAlreadyExists
	}
	return nil
}

func (sessionRunRepository *SessionRunRepository) Update(ctx context.Context, sessionRun *entities.SessionRun) error {
	return sessionRunRepository.database.WithContext(ctx).Save(sessionRun).Error
}

func (sessionRunRepository *SessionRunRepository) FailAllRunning(ctx context.Context, failureReason string, finishedAt time.Time) error {
	return sessionRunRepository.database.WithContext(ctx).Model(&entities.SessionRun{}).
		Where(clause.Eq{Column: clause.Column{Name: "status"}, Value: domains.SessionRunStatusRunning}).
		Updates(entities.SessionRun{Status: domains.SessionRunStatusFailed, FailureReason: failureReason, FinishedAt: &finishedAt}).Error
}
