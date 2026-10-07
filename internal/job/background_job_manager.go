package job

import (
	"context"

	interfaces "github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/interface"
)

type BackgroundJobManager struct {
	backgroundJobs []interfaces.IBackgroundJob
}

func NewBackgroundJobManager(backgroundJobs []interfaces.IBackgroundJob) *BackgroundJobManager {
	return &BackgroundJobManager{backgroundJobs: backgroundJobs}
}

func (backgroundJobManager *BackgroundJobManager) StartAll(ctx context.Context) {
	for _, backgroundJob := range backgroundJobManager.backgroundJobs {
		backgroundJob.Start(ctx)
	}
}
