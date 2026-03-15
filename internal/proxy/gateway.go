package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"

	"github.com/ahmed-cmyk/GopherGate/internal/config"
	"github.com/ahmed-cmyk/GopherGate/internal/middleware"
)

type routeEntry struct {
	handler http.Handler
	methods []string
}

type Gateway struct {
	routes map[string]routeEntry
}

func New(cfg *config.Config) *Gateway {
	gw := &Gateway{
		routes: make(map[string]routeEntry),
	}

	for _, route := range cfg.Routes {
		targetUrl, err := url.Parse(route.Target)
		if err != nil {
			log.Fatalf("Invalid target URL %s: %v", route.Target, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(targetUrl)

		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req) // Sets host and scheme

			if route.StripPrefix {
				// This logic is now "locked in" for this specific proxy
				if after, ok := strings.CutPrefix(req.URL.Path, route.Path); ok {
					// Force the path to be absolute
					if after == "" || after[0] != '/' {
						after = "/" + after
					}

					req.URL.Path = after
				}
			}
		}

		finalHandler := applyMiddlewares(proxy, route.Middlewares)

		gw.routes[route.Path] = routeEntry{
			handler: finalHandler,
			methods: route.Methods,
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

	if !found {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if len(matched.methods) > 0 && !slices.Contains(matched.methods, r.Method) {
		http.Error(w, "Method Not Supported", http.StatusMethodNotAllowed)
		return
	}

	matched.handler.ServeHTTP(w, r)
}

func applyMiddlewares(target http.Handler, names []string) http.Handler {
	current := target

	// Wrap from right to left so the first item in the YAML is the outermost layer
	for _, name := range names {
		if mwFunc, ok := middleware.Registry[name]; ok {
			current = mwFunc(current)
		} else {
			log.Printf("Warning: Middleware %s not found", name)
		}
	}

	return current
}
