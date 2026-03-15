package middleware

import (
	"log"
	"net"
	"net/http"
)

func getIP(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	return ip, nil
}

func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := getIP(r)
		if err != nil {
			log.Fatalf("Unable to get IP address: %v\n", err)
		}

		log.Printf("Found IP Address: %s\n", ip)

		next.ServeHTTP(w, r)
	})
}
