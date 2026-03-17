package proxy

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/ahmed-cmyk/GopherGate/internal/config"
)

type Server struct {
	Url    string
	Active bool
}

type Routes map[string][]Server

func InitBackendRoutes(routes []config.Route) *Routes {
	routesCollection := make(Routes)

	for _, route := range routes {

		var servers []Server

		for _, server := range route.Targets {
			servers = append(servers, Server{
				server,
				true,
			})
		}

		routesCollection[route.Path] = servers
	}

	return &routesCollection
}

func (rc *Routes) ScheduleRouteCheckup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		defer ticker.Stop()

		// 1. Create a stable list of keys (paths) to index into
		routes := *rc
		paths := make([]string, 0, len(routes))
		for path := range routes {
			paths = append(paths, path)
		}

		backendIndex := 0
		pathIndex := 0

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				// Refresh local ref in case it changed (optional, see note below)
				routes = *rc
				if len(paths) == 0 {
					continue
				}

				// 2. Get the current path using our index
				currentPath := paths[pathIndex]
				backends := routes[currentPath]

				if len(backends) > 0 {
					backend := backends[backendIndex]
					active := pingHealthEndpoint(backend.Url)
					if !active {
						log.Printf("Backend %s has been marked as inactive", backend.Url)
					}

					routes[currentPath][backendIndex].Active = active

					backendIndex++
				}

				// 3. Logic to move to the next backend/path
				if len(backends) == 0 || backendIndex >= len(backends) {
					pathIndex++
					backendIndex = 0
				}

				if pathIndex >= len(paths) {
					pathIndex = 0
				}

				log.Printf("Checked %s: %v", currentPath, t)
			}
		}
	}()
}

func pingHealthEndpoint(route string) bool {
	parsedUrl, err := url.JoinPath(route, "health")
	if err != nil {
		log.Printf("Failed to parse health check URL for route: %s", route)
		return false
	}

	resp, err := http.Get(parsedUrl)
	if err != nil {
		log.Printf("Failed to call health route for route %s", route)
		return false
	}

	if resp.Status != "200 OK" {
		log.Printf("Health check for route %s returned unsuccessful", route)
		return false
	}

	return true
}
