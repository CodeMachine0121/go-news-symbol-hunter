package application

import (
	"context"
	"log"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/service"
)

type SessionGradeApplication struct {
	sessionGradeService *service.SessionGradeService
}

func NewSessionGradeApplication(sessionGradeService *service.SessionGradeService) *SessionGradeApplication {
	return &SessionGradeApplication{sessionGradeService: sessionGradeService}
}

func (sessionGradeApplication *SessionGradeApplication) RunDueTradingSession(ctx context.Context) {
	// a panicking session must not stop later sessions; its run is failed on the next restart
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("trading session panicked: %v", recovered)
		}
	}()
	if err := sessionGradeApplication.sessionGradeService.RunDueTradingSession(ctx); err != nil {
		log.Printf("trading session could not be recorded: %v", err)
	}
}

func (sessionGradeApplication *SessionGradeApplication) GetTrackedSymbolGrades(ctx context.Context, getTrackedSymbolGradesDto dto.GetTrackedSymbolGradesDto) ([]dto.TrackedSymbolGradeDto, error) {
	return sessionGradeApplication.sessionGradeService.GetTrackedSymbolGrades(ctx, getTrackedSymbolGradesDto)
}

func (sessionGradeApplication *SessionGradeApplication) FailInterruptedSessionRuns(ctx context.Context) error {
	return sessionGradeApplication.sessionGradeService.FailInterruptedSessionRuns(ctx)
}
