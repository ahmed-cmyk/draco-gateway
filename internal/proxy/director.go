package proxy

import (
	"net/http"
	"strings"

	config "github.com/ahmed-cmyk/GopherGate/internal"
)

func ApplyDirector(route *config.Route, originalDirector func(*http.Request)) func(*http.Request) {
	return func(req *http.Request) {
		originalDirector(req) // Sets host and scheme

		// Remove unwanted headers
		for _, h := range route.Headers.Remove {
			req.Header.Del(h)
		}

		// Set new headers
		for key, value := range route.Headers.Set {
			req.Header.Set(key, value)
		}

		if route.StripPrefix {
			// This logic is now "locked in" for this specific proxy
			if after, ok := strings.CutPrefix(req.URL.Path, route.Path); ok {
				// Force the path to be absolute
				if !strings.HasPrefix(after, "/") {
					after = "/" + after
				}

				req.URL.Path = after
			}
		}
	}
}
