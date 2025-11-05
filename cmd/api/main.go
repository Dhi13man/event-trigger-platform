package main

import (
	"log"

	"github.com/dhima/event-trigger-platform/internal/api"
)

func main() {
	srv := api.NewServer()
	if err := srv.Serve(); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}
