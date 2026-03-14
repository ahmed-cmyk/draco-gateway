package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahmed-cmyk/GopherGate/internal/config"
)

func main() {
	// Create a context that is cancelled when SIGINT or SIGNTERM is received
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var config config.Config

	err := config.LoadData("config.yaml")
	if err != nil {
		log.Fatalf("Error unmarshaling YAML: %v\n", err)
		os.Exit(1)
	}

	remote, err := url.Parse("https://api.chucknorris.io")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)

	port := fmt.Sprintf(":%s", config.Server.Port)
	srv := &http.Server{
		Addr:         port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Handler:      proxy,
	}

	// Run server inside a goroutine so that it doesn't block
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	log.Printf("Starting service: %s\n", config.ServiceName)
	log.Printf("Listening on port %s\n", config.Server.Port)

	// Wait for the interrupt signal
	<-ctx.Done()

	log.Println("Shutting down server gracefully...")
	log.Println("Server gracefully stopped")
}
