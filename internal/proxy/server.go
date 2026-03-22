package proxy

import (
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

type Server struct {
	URL      string
	Attempts int
	Alive    bool
	mu       sync.Mutex
}

func (s *Server) GetURL() string {
	return s.URL
}

func (s *Server) SetStatus(alive bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if alive {
		s.Attempts = 0
		if !s.Alive {
			s.Alive = true
			log.Infof("Server %s is BACK ONLINE", s.URL)
		}
		return
	}

	s.Attempts++
	if s.Attempts >= 3 && s.Alive {
		s.Alive = false
		s.Attempts = 0 // Optional: reset so it can start the 3-strike count again if it blips back
		log.Warnf("Server %s is now officially DEAD", s.URL)
	}
}

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
