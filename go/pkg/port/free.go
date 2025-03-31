package port

import (
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
)

// FreePort provides utilities for finding available network ports.
// It manages a pool of assigned ports to prevent the same port from
// being returned multiple times within the same process.
type FreePort struct {
	mu       sync.RWMutex
	min      int
	max      int
	attempts int

	// The caller may request multiple ports without binding them immediately
	// so we need to keep track of which ports are assigned.
	assigned map[int]bool
}

// New creates a new FreePort instance for finding available ports.
// The returned instance keeps track of ports it has assigned to prevent
// returning the same port twice, even if the actual binding hasn't occurred.
//
// By default, ports are selected from the range 10000-65535, which falls
// within the standard range for ephemeral/private ports.
//
// Example:
//
//	// Create a new port finder
//	portFinder := port.New()
//
//	// Get multiple available ports
//	httpPort := portFinder.Get()
//	grpcPort := portFinder.Get()
//	metricsPort := portFinder.Get()
//
//	fmt.Printf("Running HTTP on port %d, gRPC on port %d, metrics on port %d\n",
//	    httpPort, grpcPort, metricsPort)
func New() *FreePort {
	return &FreePort{
		min:      10000,
		max:      65535,
		attempts: 10,
		assigned: map[int]bool{},
		mu:       sync.RWMutex{},
	}
}

// Get returns an available TCP port number.
// The port is guaranteed to be available at the time of the call,
// and will not be returned again by the same FreePort instance.
//
// This method will attempt to find an available port by:
// 1. Selecting a random port in the range 10000-65535
// 2. Checking that the port hasn't already been assigned by this instance
// 3. Verifying availability by attempting to bind to 127.0.0.1 on that port
// 4. Marking the port as assigned to prevent future reuse
//
// If no available port can be found after multiple attempts, Get will panic.
// For cases where error handling is preferred over panicking, use GetWithError.
//
// Example:
//
//	finder := port.New()
//	serverPort := finder.Get()
//
//	// Start your server on this port
//	server := &http.Server{
//	    Addr:    fmt.Sprintf(":%d", serverPort),
//	    Handler: mux,
//	}
//	server.ListenAndServe()
func (f *FreePort) Get() int {
	port, err := f.GetWithError()
	if err != nil {
		panic(err)
	}

	return port
}

// GetWithError returns an available TCP port number or an error if no port
// could be found after multiple attempts.
//
// This method works the same as Get() but returns an error instead of
// panicking when no available ports can be found. This is preferred in
// production code where error handling is more appropriate than panicking.
//
// Example:
//
//	finder := port.New()
//	port, err := finder.GetWithError()
//	if err != nil {
//	    log.Fatalf("Failed to find available port: %v", err)
//	}
//
//	// Use the port
//	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
func (f *FreePort) GetWithError() (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := 0; i < f.attempts; i++ {

		// nolint:gosec
		// This isn't cryptography
		port := rand.IntN(f.max-f.min) + f.min
		if f.assigned[port] {
			continue
		}

		ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: port, Zone: ""})
		if err != nil {
			continue
		}
		err = ln.Close()
		if err != nil {
			return -1, err
		}
		f.assigned[port] = true
		return port, nil
	}
	return -1, fmt.Errorf("could not find a free port, maybe increase attempts?")
}
