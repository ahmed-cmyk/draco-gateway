package health

import (
	"context"
	"net/http"
	"time"
)

type Target interface {
	GetURL() string
	SetStatus(alive bool)
}

type Pinger struct {
	client *http.Client
}

func NewPinger(timeout time.Duration) *Pinger {
	return &Pinger{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *Pinger) Ping(ctx context.Context, target Target) {
	req, err := http.NewRequestWithContext(ctx, "GET", target.GetURL(), nil)
	if err != nil {
		target.SetStatus(false)
		return
	}

	resp, err := p.client.Do(req)

	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		target.SetStatus(false)
		return
	}

	defer resp.Body.Close()
	target.SetStatus(true)
}
