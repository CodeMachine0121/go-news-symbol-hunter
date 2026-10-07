package interfaces

import "context"

type IBackgroundJob interface {
	// returns immediately; the job keeps running until ctx is cancelled
	Start(ctx context.Context)
}
