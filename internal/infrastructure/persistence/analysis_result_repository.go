package persistence

import (
	"context"
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
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

func (analysisResultRepository *AnalysisResultRepository) FindForBacktest(ctx context.Context, gradeBacktestFilter vo.GradeBacktestFilterVo, limit int) ([]entities.AnalysisResult, error) {
	query := analysisResultRepository.database.WithContext(ctx)
	if !gradeBacktestFilter.CompletedPeriod.From.IsZero() {
		query = query.Where(clause.Gte{Column: clause.Column{Name: "created_at"}, Value: gradeBacktestFilter.CompletedPeriod.From})
	}
	if !gradeBacktestFilter.CompletedPeriod.Before.IsZero() {
		query = query.Where(clause.Lt{Column: clause.Column{Name: "created_at"}, Value: gradeBacktestFilter.CompletedPeriod.Before})
	}
	if gradeBacktestFilter.Symbol != "" {
		query = query.Where(clause.Eq{Column: clause.Column{Name: "symbol"}, Value: gradeBacktestFilter.Symbol}).
			Where(clause.Eq{Column: clause.Column{Name: "category"}, Value: gradeBacktestFilter.Category})
	}
	analysisResults := []entities.AnalysisResult{}
	err := query.Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}}).Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}}).Limit(limit).Find(&analysisResults).Error
	if err != nil {
		return nil, err
	}
	return analysisResults, nil
}
