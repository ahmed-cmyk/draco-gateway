package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahmed-cmyk/GopherGate/internal/config"
	"github.com/ahmed-cmyk/GopherGate/internal/middleware"
	"github.com/ahmed-cmyk/GopherGate/internal/proxy"
	"golang.org/x/time/rate"
)

func main() {
	// Create a context that is cancelled when SIGINT or SIGNTERM is received
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cfg config.Config

	err := cfg.LoadData("config.yaml")
	if err != nil {
		log.Fatalf("Error unmarshaling YAML: %v\n", err)
	}

	// Setup Gateway Instance
	gateway := setupGateway(&cfg)

	port := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:         port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Handler:      gateway,
	}

	// Run server inside a goroutine so that it doesn't block
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	log.Printf("Starting service: %s\n", cfg.ServiceName)
	log.Printf("Listening on port %s\n", cfg.Server.Port)

	// Wait for the interrupt signal
	<-ctx.Done()

	log.Println("Shutting down server gracefully...")
	log.Println("Server gracefully stopped")
}

func setupGateway(cfg *config.Config) *proxy.Gateway {
	// Initialize the stateful logic
	limiterManager := middleware.NewLimiter(rate.Every(time.Minute), 5)

	// Prime the middleware with its manager
	rateLimitMW := middleware.RateLimit(&limiterManager)

	// Register it dynamically
	middleware.Registry["rate_limit"] = rateLimitMW

	return proxy.New(cfg)
}
