package service

import (
	"context"
	"log/slog"
	"time"

	"voca-test/internal/repository"
)

// WorkerService handles background tasks
type WorkerService struct {
	iRepo repository.IdempotencyRepository
}

func NewWorkerService(iRepo repository.IdempotencyRepository) *WorkerService {
	return &WorkerService{
		iRepo: iRepo,
	}
}

// StartCleanupWorker starts a background goroutine that cleans up old idempotency keys
func (s *WorkerService) StartCleanupWorker(ctx context.Context, interval time.Duration, retention time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("Idempotency cleanup worker started", 
		"interval", interval.String(), 
		"retention", retention.String())

	for {
		select {
		case <-ticker.C:
			count, err := s.iRepo.DeleteExpiredKeys(ctx, retention)
			if err != nil {
				slog.Error("Failed to cleanup expired idempotency keys", "error", err)
			} else if count > 0 {
				slog.Info("Cleaned up expired idempotency keys", "count", count)
			}
		case <-ctx.Done():
			slog.Info("Idempotency cleanup worker stopping")
			return
		}
	}
}
