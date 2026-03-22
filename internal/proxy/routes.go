package proxy

import (
	"sync"

	config "github.com/ahmed-cmyk/GopherGate/internal"
	"github.com/ahmed-cmyk/GopherGate/internal/health"
)

type Routes struct {
	route map[string][]Server
	mu    sync.RWMutex
}

func InitRoutes(routes *[]config.Route) *Routes {
	r := &Routes{
		route: make(map[string][]Server),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, route := range *routes {
		var servers []Server

		for _, target := range route.Targets {
			servers = append(servers, Server{
				URL:   target,
				Alive: true,
			})
		}

		r.route[route.Path] = servers
	}

	return r
}

func (r *Routes) GetPaths() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.route))
	for p := range r.route {
		paths = append(paths, p)
	}

	return paths
}

func (r *Routes) GetTargets(path string) []health.Target {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := r.route[path]
	targets := make([]health.Target, len(servers))
	for i := range servers {
		targets[i] = &servers[i]
	}

	return targets
}
