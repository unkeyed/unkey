---
title: prometheus
description: "provides HTTP server infrastructure for exposing Prometheus metrics"
---

Package prometheus provides HTTP server infrastructure for exposing Prometheus metrics.

This package is the entry point for running a metrics server that exposes the /metrics endpoint for Prometheus scraping. The actual metric collectors are defined in the \[metrics] subpackage.

### Usage

Start the metrics server on a dedicated port, typically in a goroutine since \[Serve] blocks until the server stops or encounters an error:

	go func() {
	    if err := prometheus.Serve(":9090"); err != nil {
	        log.Printf("metrics server error: %v", err)
	    }
	}()

### Architecture

The package is split into two parts:

  - This package: HTTP server that exposes metrics at GET /metrics
  - \[metrics] subpackage: All metric collectors organized by subsystem

This separation allows services to import only the metrics they need without pulling in HTTP server dependencies.

Package prometheus provides utilities for exposing Prometheus metrics over HTTP.

This package makes it easy to integrate Prometheus metrics collection and exposure into applications built with the zen framework. It handles the setup of a metrics endpoint that Prometheus can scrape to collect runtime metrics from your application.

Common use cases include:

  - Creating a dedicated metrics server separate from your application server
  - Adding metrics endpoints to existing zen-based applications
  - Setting up consistent metrics across multiple microservices

This package is designed to work seamlessly with the zen framework while providing all the functionality of the standard Prometheus client library.

## Functions

### func New

```go
func New() (*zen.Server, error)
```

New creates a zen server that exposes Prometheus metrics at the /metrics endpoint. The server is configured to handle GET requests to the /metrics path using the standard Prometheus HTTP handler, which serves metrics in a format that Prometheus can scrape.

New is used to create a standalone metrics server that can be started separately from your main application server, which is a common pattern for microservices architectures where concerns are separated.

Parameters:

  - config: Configuration for the server, including required dependencies.

Returns:

  - A configured zen.Server ready to be started.
  - An error if server creation fails, typically due to invalid configuration.

Example usage:

	// Create a dedicated metrics server
	server, err := prometheus.New(prometheus.Config{
	   ,
	})
	if err != nil {
	    log.Fatalf("Failed to create metrics server: %v", err)
	}

	// Start the metrics server on port 9090
	go func() {
	    if err := server.Listen(":9090"); err != nil {
	        log.Fatalf("Metrics server failed: %v", err)
	    }
	}()

When used with CLI commands, the server can be started with a command like:

	myapp metrics --port=9090

See \[zen.New] for details on the underlying server creation. See \[promhttp.Handler] for details on the Prometheus metrics handler.

### func Serve

```go
func Serve(addr string) error
```

Serve starts a simple HTTP server that exposes Prometheus metrics at GET /metrics. The server listens on the provided address (e.g., ":9090" or "127.0.0.1:9090").

This is a simpler alternative to New() that doesn't require the zen framework. It blocks until the server stops or an error occurs.

Example usage:

	go func() {
	    if err := prometheus.Serve(":9090"); err != nil {
	        log.Fatalf("Metrics server failed: %v", err)
	    }
	}()

