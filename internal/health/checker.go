package health

import (
	"context"
	"time"

	"github.com/charmbracelet/log"
)

func NewHealthChecker(routes RouteStore, interval time.Duration, timeout time.Duration) *HealthChecker {
	pinger := NewPinger(timeout)

	return &HealthChecker{
		routes:   routes,
		interval: interval,
		pinger:   pinger,
	}
}

func (hc *HealthChecker) StartHealthChecker(ctx context.Context) {
	healthTicker := time.NewTicker(hc.interval)
	defer healthTicker.Stop()

	for {
		select {
		case <-healthTicker.C:
			paths := hc.routes.GetPaths()

			// TODO: Check health of all servers
			for _, path := range paths {
				servers := hc.routes.GetTargets(path)
				for _, server := range servers {
					// Launch each ping in its own goroutine
					go func(s Target) {
						log.Debugf("Pinging: %s", s.GetURL())
						hc.pinger.Ping(ctx, s)
					}(server)
				}
			}
		case <-ctx.Done():
			log.Info("Health checker stopped")
			return
		}
	}
}
