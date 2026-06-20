package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/saaof/order-platform/catalog-service/internal/application"
)

func RunReservationExpiration(ctx context.Context, service *application.Service, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			count, err := service.ExpirePending(ctx, now, 100)
			if err != nil {
				slog.Error("reservation expiration failed", "error", err)
				continue
			}
			if count > 0 {
				slog.Info("expired stock reservations", "count", count)
			}
		}
	}
}
