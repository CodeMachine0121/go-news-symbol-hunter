package persistence

import (
	"context"
	"errors"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GradeRecordRepository struct {
	database *gorm.DB
}

func NewGradeRecordRepository(database *gorm.DB) *GradeRecordRepository {
	return &GradeRecordRepository{database: database}
}

func (gradeRecordRepository *GradeRecordRepository) Create(ctx context.Context, gradeRecord *entities.GradeRecord) error {
	return gradeRecordRepository.database.WithContext(ctx).Create(gradeRecord).Error
}

func (gradeRecordRepository *GradeRecordRepository) FindByAnalysisEventID(ctx context.Context, analysisEventID uint) (*entities.GradeRecord, error) {
	var gradeRecord entities.GradeRecord
	err := gradeRecordRepository.database.WithContext(ctx).Where(clause.Eq{Column: clause.Column{Name: "analysis_event_id"}, Value: analysisEventID}).First(&gradeRecord).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &gradeRecord, nil
}

// one indexed lookup per symbol, so the cost follows the number of tracked symbols rather than
// how many background records have piled up
func (gradeRecordRepository *GradeRecordRepository) FindLatestBackground(ctx context.Context, symbols []vo.SymbolVo) ([]entities.GradeRecord, error) {
	gradeRecords := []entities.GradeRecord{}
	for _, symbol := range symbols {
		var gradeRecord entities.GradeRecord
		err := gradeRecordRepository.database.WithContext(ctx).
			Where(clause.Eq{Column: clause.Column{Name: "source"}, Value: vo.GradeSourceBackground}).
			Where(clause.Eq{Column: clause.Column{Name: "symbol"}, Value: symbol.Value}).
			Where(clause.Eq{Column: clause.Column{Name: "category"}, Value: symbol.Category.Value}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "trading_day"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: true}).
			Take(&gradeRecord).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		gradeRecords = append(gradeRecords, gradeRecord)
	}
	return gradeRecords, nil
}

func (gradeRecordRepository *GradeRecordRepository) FindForBacktest(ctx context.Context, gradeBacktestFilter vo.GradeBacktestFilterVo, limit int) ([]entities.GradeRecord, error) {
	// a manual record is saved before its event is marked succeeded, so a failed status update can leave a
	// record behind an analysis that did not succeed
	succeededAnalysisEventIDs := gradeRecordRepository.database.WithContext(ctx).Model(&entities.AnalysisEvent{}).Select("id").
		Where(clause.Eq{Column: clause.Column{Name: "status"}, Value: domains.AnalysisStatusSucceeded})
	query := gradeRecordRepository.database.WithContext(ctx).Where(clause.Or(
		clause.Eq{Column: clause.Column{Name: "source"}, Value: vo.GradeSourceBackground},
		// nested so gorm brackets the subquery; a lone value would be rendered as "= SELECT ..."
		clause.IN{Column: clause.Column{Name: "analysis_event_id"}, Values: []any{[]any{succeededAnalysisEventIDs}}},
	))
	if gradeBacktestFilter.Source != "" {
		query = query.Where(clause.Eq{Column: clause.Column{Name: "source"}, Value: gradeBacktestFilter.Source})
	}
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
	gradeRecords := []entities.GradeRecord{}
	err := query.Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}}).Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}}).Limit(limit).Find(&gradeRecords).Error
	if err != nil {
		return nil, err
	}
	return gradeRecords, nil
}
