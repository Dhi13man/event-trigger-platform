package api

import (
	"log"
	"net/http"
	"os"
)

// Server orchestrates HTTP routing and dependencies for the API service.
type Server struct {
	port string
}

// NewServer wires the API dependencies together.
func NewServer() *Server {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	return &Server{port: port}
}

// Serve starts the HTTP server and blocks until error or shutdown.
func (s *Server) Serve() error {
	// Stub handlers - replace with real routing (Gin/Echo) during implementation
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Event Trigger Platform API - stub implementation"}`))
	})

	addr := ":" + s.port
	log.Printf("API server listening on %s", addr)

	// This blocks until the server crashes or is shut down
	return http.ListenAndServe(addr, nil)
}
