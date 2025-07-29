package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type HelloResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	// Hello endpoint
	mux.HandleFunc("/v1/hello", func(w http.ResponseWriter, r *http.Request) {
		response := HelloResponse{
			Message:   "Hello from demo API",
			Timestamp: time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// OpenAPI spec endpoint
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		spec := `openapi: 3.0.3
info:
  title: Demo API
  description: A simple demo API for testing deployments
  version: 1.0.0
  contact:
    name: Unkey Support
    email: support@unkey.com
servers:
  - url: /v1
    description: API v1
paths:
  /liveness:
    get:
      operationId: getLiveness
      summary: Health check endpoint
      description: Returns OK if the service is healthy
      responses:
        '200':
          description: Service is healthy
          content:
            text/plain:
              schema:
                type: string
                example: OK
  /hello:
    get:
      operationId: getHello
      summary: Hello endpoint
      description: Returns a greeting message with timestamp
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                type: object
                properties:
                  message:
                    type: number
                    example: Hello from demo API
                  timestamp:
                    type: string
                    format: date-time
                    example: 2023-12-07T10:30:00Z
                required:
                  - message
                  - timestamp
components:
  schemas:
    HelloResponse:
      type: object
      properties:
        message:
          type: string
          description: The greeting message
        timestamp:
          type: string
          format: date-time
          description: The current timestamp
      required:
        - message
        - timestamp`

		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, spec)
	})

	log.Printf("Demo API starting on port %s", port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
