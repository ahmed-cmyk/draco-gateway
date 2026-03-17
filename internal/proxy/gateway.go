package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"

	"github.com/ahmed-cmyk/GopherGate/internal/config"
	loadbalancer "github.com/ahmed-cmyk/GopherGate/internal/loadBalancer"
	"github.com/ahmed-cmyk/GopherGate/internal/middleware"
)

type routeEntry struct {
	balancer loadbalancer.Balancer
	handler  http.Handler
	methods  []string
}

type Gateway struct {
	routes map[string]routeEntry
}

func New(cfg *config.Config, routeMap *Routes) *Gateway {
	gw := &Gateway{
		routes: make(map[string]routeEntry),
	}

	for _, route := range cfg.Routes {
		targetUrl, err := url.Parse(route.Targets[0])
		if err != nil {
			log.Printf("Invalid target URL %s: %v", route.Targets[0], err)
		}

		proxy := httputil.NewSingleHostReverseProxy(targetUrl)

		originalDirector := proxy.Director
		// Create a new copy of "route" scoped to this loop iteration
		route := route
		proxy.Director = ApplyDirector(&route, originalDirector)

		finalHandler := applyMiddlewares(proxy, route.Middlewares)

		var servers []string

		for _, server := range (*routeMap)[route.Path] {
			servers = append(servers, server.Url)
		}

		gw.routes[route.Path] = routeEntry{
			balancer: middleware.ResolveBalancer(route.Path, route.Balancer, servers),
			handler:  finalHandler,
			methods:  route.Methods,
		}
	}
	return gw
}

func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var found bool
	var matched routeEntry

	for path, entry := range gw.routes {
		if strings.HasPrefix(r.URL.Path, path) {
			matched = entry
			found = true
			break
		}
	}

	// If no route has been found return a 404 error
	if !found {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Check if method is supported
	if len(matched.methods) > 0 && !slices.Contains(matched.methods, r.Method) {
		http.Error(w, "Method Not Supported", http.StatusMethodNotAllowed)
		return
	}

	// Select next backend for the matched route, if nothing matches return 500 error
	host, err := matched.balancer.NextBackend()
	if err != nil {
		http.Error(w, "Server Error", 500)
		return
	}

	r.URL.Host = string(host)
	r.URL.Scheme = "http"
	r.Host = string(host)

	log.Printf("Routing %s request to %s", r.URL.Path, host)

	matched.handler.ServeHTTP(w, r)
}

func applyMiddlewares(target http.Handler, names []string) http.Handler {
	current := target

	for _, name := range names {
		if mwFunc, ok := middleware.Registry[name]; ok {
			current = mwFunc(current)
		} else {
			log.Printf("Warning: Middleware %s not found", name)
		}
	}

	return current
}
