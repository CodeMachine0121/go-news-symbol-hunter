package application

import (
	"context"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
)

type GradeBacktestApplication struct {
	gradeBacktestService *service.GradeBacktestService
}

func NewGradeBacktestApplication(gradeBacktestService *service.GradeBacktestService) *GradeBacktestApplication {
	return &GradeBacktestApplication{gradeBacktestService: gradeBacktestService}
}

func (gradeBacktestApplication *GradeBacktestApplication) GetGradeBacktest(ctx context.Context, getGradeBacktestDto dto.GetGradeBacktestDto) (dto.GradeBacktestDto, error) {
	return gradeBacktestApplication.gradeBacktestService.GetGradeBacktest(ctx, getGradeBacktestDto)
}
