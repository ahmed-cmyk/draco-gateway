package health

import (
	"time"
)

type RouteStore interface {
	GetPaths() []string
	GetTargets(path string) []Target
}

type HealthChecker struct {
	routes   RouteStore
	interval time.Duration
	pinger   *Pinger
}
