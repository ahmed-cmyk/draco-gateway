package proxy

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

func StartServer(port string, gateway http.Handler) {
	server := &http.Server{
		Addr:         port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Handler:      gateway,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Errorf("Failed to start server: %v\n", err)
	}
}
