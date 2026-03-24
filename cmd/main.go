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
	"github.com/joho/godotenv"
)

func main() {
	// Create a context that is cancelled when SIGINT or SIGNTERM is received
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env file")
		return
	}

	var cfg config.Config

	// Load configuration data from "config.yaml" and throw an error if it fails
	err = cfg.LoadData("config.yaml")
	if err != nil {
		log.Errorf("Error unmarshaling YAML: %v\n", err)
	}

	// Set log level to debug so that we can see debug logs
	log.SetLevel(log.DebugLevel)

	log.Debug("Starting routes setup")
	routes := proxy.InitRoutes(&cfg.Routes)
	log.Debug("Completed routes setup")

	// Register Rate Limiter instance
	middleware.RegisterRateLimiter(time.Second, 50)

	// Instantiate Gateway object
	gateway := proxy.NewGateway(&cfg, routes)
	port := fmt.Sprintf(":%s", cfg.Server.Port)

	// Set HealthChecker duration between requests
	interval := time.Duration(5 * time.Second)
	// Set HealthChecker timeout duration
	timeout := time.Duration(30 * time.Second)

	// Start the health checker
	healthChecker := health.NewHealthChecker(routes, interval, timeout)

	// Wire up health status changes to update the load balancer
	healthChecker.OnStatusChange(func(url string, healthy bool) {
		gateway.UpdateBackendHealth(url, healthy)
	})

	log.Debug("Starting HealthChecker")
	go healthChecker.StartHealthChecker(ctx)
	log.Debug("Started HealthChecker")

	middleware.InitKey(os.Getenv("JWT_KEY"))

	go proxy.StartServer(port, gateway)

	log.Infof("Starting service: %s\n", cfg.ServiceName)
	log.Infof("Listening on port %s\n", cfg.Server.Port)

	// Wait for the interrupt signal
	<-ctx.Done()

	log.Infof("Route ticker stopped")
	log.Infof("Shutting down server gracefully...")
	log.Infof("Server gracefully stopped")
}
