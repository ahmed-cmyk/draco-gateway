package proxy

import "sync"

type Routes struct {
	route map[string][]Server
	mu    sync.RWMutex
}

type Server struct {
	URL    string
	Active bool
}
