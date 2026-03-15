package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/ahmed-cmyk/GopherGate/internal/config"
	"github.com/ahmed-cmyk/GopherGate/internal/middleware"
)

type Gateway struct {
	handlers map[string]http.Handler
}

func New(cfg *config.Config) *Gateway {
	gw := &Gateway{
		handlers: make(map[string]http.Handler),
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

		finalHandler := ApplyMiddlewares(proxy, route.Middlewares)

		gw.handlers[route.Path] = finalHandler
	}
	return gw
}

func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	incomingPath := r.URL.Path
	var matchedHandler http.Handler

	for path, proxy := range gw.handlers {
		if strings.HasPrefix(incomingPath, path) {
			matchedHandler = proxy
			break
		}
	}

	if matchedHandler == nil {
		http.Error(w, "Not Found", 404)
		return
	}

	matchedHandler.ServeHTTP(w, r)
}

func ApplyMiddlewares(target http.Handler, names []string) http.Handler {
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
