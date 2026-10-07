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

type SessionRunRepository struct {
	database *gorm.DB
}

func NewSessionRunRepository(database *gorm.DB) *SessionRunRepository {
	return &SessionRunRepository{database: database}
}

func (sessionRunRepository *SessionRunRepository) Create(ctx context.Context, sessionRun *entities.SessionRun) error {
	err := sessionRunRepository.database.WithContext(ctx).Create(sessionRun).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return service.ErrSessionRunAlreadyExists
	}
	return err
}

func (sessionRunRepository *SessionRunRepository) Update(ctx context.Context, sessionRun *entities.SessionRun) error {
	return sessionRunRepository.database.WithContext(ctx).Save(sessionRun).Error
}

func (sessionRunRepository *SessionRunRepository) FailAllRunning(ctx context.Context, failureReason string, finishedAt time.Time) error {
	return sessionRunRepository.database.WithContext(ctx).Model(&entities.SessionRun{}).
		Where(clause.Eq{Column: clause.Column{Name: "status"}, Value: domains.SessionRunStatusRunning}).
		Updates(entities.SessionRun{Status: domains.SessionRunStatusFailed, FailureReason: failureReason, FinishedAt: &finishedAt}).Error
}
