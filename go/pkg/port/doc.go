// Package port provides utilities for finding and managing available network ports.
//
// This package is particularly useful for testing scenarios where multiple
// services need to run on unique ports without conflicting with each other
// or with existing services. It safely locates available ports through actual
// network binding and offers mechanisms to track allocated ports to prevent
// reuse within the same process.
//
// The implementation uses a combination of random port selection and actual
// TCP socket binding to verify availability. This approach is more reliable
// than just checking if a port is currently in use, as it accounts for
// ports that may be temporarily unavailable or restricted by the operating system.
//
// Basic usage:
//
//	// Create a port finder
//	finder := port.New()
//
//	// Get an available port
//	port := finder.Get()
//
//	// Use the port for your service
//	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
//
//	// Or for testing multiple services:
//	port1 := finder.Get()
//	port2 := finder.Get()
//	port3 := finder.Get()
//
// The package tracks ports it has assigned within the current process
// to ensure the same port isn't returned twice, even if the port hasn't
// been bound yet.
//
// Note that port availability is only guaranteed at the moment Get() is called.
// If there is a delay between getting the port and binding to it, another
// process could potentially bind to that port in the meantime.
package port
