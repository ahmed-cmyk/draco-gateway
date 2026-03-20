package proxy

import config "github.com/ahmed-cmyk/GopherGate/internal"

func SetRoutes(routes *[]config.Route) *Routes {
	r := &Routes{
		route: make(map[string][]Server),
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range *routes {
		var servers []Server

		for _, target := range route.Targets {
			servers = append(servers, Server{
				URL:    target,
				Active: true,
			})
		}

		r.route[route.Path] = servers
	}

	return r
}
