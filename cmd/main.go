package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ahmed-cmyk/GopherGate/internal/config"
)

func main() {
	var config config.Config

	data, err := config.CheckConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	err = config.LoadData(data)
	if err != nil {
		fmt.Printf("Error unmarshaling YAML: %v\n", err)
		os.Exit(1)
	}

	port := fmt.Sprintf(":%s", config.Server.Port)

	fmt.Printf("Running on port %s\n", config.Server.Port)

	err = http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
