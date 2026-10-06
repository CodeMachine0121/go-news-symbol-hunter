package persistence

import (
	"context"
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AnalysisResultRepository struct {
	database *gorm.DB
}

func NewAnalysisResultRepository(database *gorm.DB) *AnalysisResultRepository {
	return &AnalysisResultRepository{database: database}
}

func (analysisResultRepository *AnalysisResultRepository) Create(ctx context.Context, analysisResult *entities.AnalysisResult) error {
	return analysisResultRepository.database.WithContext(ctx).Create(analysisResult).Error
}

func (analysisResultRepository *AnalysisResultRepository) FindByAnalysisEventID(ctx context.Context, analysisEventID uint) (*entities.AnalysisResult, error) {
	var analysisResult entities.AnalysisResult
	err := analysisResultRepository.database.WithContext(ctx).Where(clause.Eq{Column: clause.Column{Name: "analysis_event_id"}, Value: analysisEventID}).First(&analysisResult).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &analysisResult, nil
}
