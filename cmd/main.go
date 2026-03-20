package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/ahmed-cmyk/GopherGate/internal"
	"github.com/ahmed-cmyk/GopherGate/internal/health"
	"github.com/ahmed-cmyk/GopherGate/internal/middleware"
	"github.com/ahmed-cmyk/GopherGate/internal/proxy"
	"github.com/charmbracelet/log"
	"golang.org/x/time/rate"
)

func main() {
	// Create a context that is cancelled when SIGINT or SIGNTERM is received
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cfg config.Config

	err := cfg.LoadData("config.yaml")
	if err != nil {
		log.Errorf("Error unmarshaling YAML: %v\n", err)
	}

	routes := proxy.SetRoutes(&cfg.Routes)

	// Setup Gateway Instance
	gateway := setupGateway(&cfg, routes)
	port := fmt.Sprintf(":%s", cfg.Server.Port)

	go proxy.StartServer(port, gateway)

	// Start the health checker
	duration := time.Duration(5) * time.Second
	healthChecker := health.NewHealthChecker(routes, &duration)
	go healthChecker.StartHealthChecker(ctx)

	log.Infof("Starting service: %s\n", cfg.ServiceName)
	log.Infof("Listening on port %s\n", cfg.Server.Port)

	// Wait for the interrupt signal
	<-ctx.Done()

	log.Infof("Route ticker stopped")

	log.Infof("Shutting down server gracefully...")
	log.Infof("Server gracefully stopped")
}

func setupGateway(cfg *config.Config, routeMap *proxy.Routes) *proxy.Gateway {
	// Initialize the stateful logic
	limiterManager := middleware.NewLimiter(rate.Every(time.Minute), 50)

	// Prime the middleware with its manager
	rateLimitMW := middleware.RateLimit(&limiterManager)

	// Register it dynamically
	middleware.Registry["rate_limit"] = rateLimitMW

	return proxy.NewGateway(cfg, routeMap)
}
