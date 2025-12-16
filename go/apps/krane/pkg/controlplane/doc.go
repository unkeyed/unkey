// Package controlplane provides client components for connecting to the Krane control plane
// service. It implements authentication, request interception, and event streaming
// capabilities required for distributed deployment coordination.
//
// The package is designed to work with Connect RPC and provides both synchronous
// client creation and asynchronous event watching functionality. All requests are
// automatically authenticated with bearer tokens and annotated with regional and
// shard metadata for proper routing within the distributed system.
//
// # Key Types
//
// The main entry point is [NewClient] which creates a configured [ClientConfig]
// for control plane communication. For event streaming, use [NewWatcher] to create
// a [Watcher] that can handle both live and synthetic event streams.
//
// # Usage
//
// Creating a control plane client:
//
//	cfg := controlplane.ClientConfig{
//	    URL:         "https://controlplane.example.com",
//	    BearerToken: "your-auth-token",
//	    Region:      "us-west-2",
//	    Shard:       "primary",
//	}
//	client := controlplane.NewClient(cfg)
//
// Setting up an event watcher:
//
//	watcherCfg := controlplane.WatcherConfig[*YourEvent]{
//	    Logger:       logger,
//	    InstanceID:   "node-123",
//	    Region:      "us-west-2",
//	    Shard:       "primary",
//	    CreateStream: client.Watch,
//	}
//	watcher := controlplane.NewWatcher(watcherCfg)
//
//	// Start watching for live events
//	go watcher.Watch(ctx)
//
//	// Consume events
//	eventCh := watcher.Consume()
//	for event := range eventCh {
//	    processEvent(event)
//	}
//
// # Authentication and Metadata
//
// All outgoing requests are automatically enhanced with authentication headers
// (Authorization bearer token) and routing metadata (X-Krane-Region, X-Krane-Shard)
// through an interceptor that works for both unary and streaming RPC calls.
//
// # Error Handling
//
// The watcher implements exponential backoff with jitter for reconnection attempts
// and provides separate methods for synchronization (periodic full sync) and live
// streaming (real-time event delivery). Connection failures are logged and automatic
// reconnection is attempted with increasing intervals.
package controlplane
