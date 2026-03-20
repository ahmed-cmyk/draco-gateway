package health

import (
	"context"
	"time"

	"github.com/ahmed-cmyk/GopherGate/internal/proxy"
	"github.com/charmbracelet/log"
)

type HealthChecker struct {
	routes   *proxy.Routes
	interval *time.Duration
}

func NewHealthChecker(routes *proxy.Routes, interval *time.Duration) *HealthChecker {
	return &HealthChecker{
		routes:   routes,
		interval: interval,
	}
}

func (hc *HealthChecker) StartHealthChecker(ctx context.Context) {
	healthTicker := time.NewTicker(*hc.interval)
	defer healthTicker.Stop()

	for {
		select {
		case <-healthTicker.C:
			log.Debug("Checking health: ", time.Now())
			// TODO: Check health of all servers
		case <-ctx.Done():
			log.Debug("Health checker stopped")
			return
		}
	}
}
