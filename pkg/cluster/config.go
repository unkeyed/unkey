package cluster

import clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"

// Config configures a gossip cluster node.
type Config struct {
	// Region identifies the geographic region (e.g. "us-east-1").
	Region string

	// NodeID is a unique identifier for this instance.
	NodeID string

	// BindAddr is the address to bind memberlist listeners on. Default "0.0.0.0".
	BindAddr string

	// BindPort is the LAN memberlist port. Default 7946.
	BindPort int

	// WANBindPort is the WAN memberlist port (used when this node becomes gateway). Default 7947.
	WANBindPort int

	// LANSeeds are addresses of existing LAN cluster members to join (e.g. k8s headless service).
	LANSeeds []string

	// WANSeeds are addresses of cross-region gateways to join.
	WANSeeds []string

	// SecretKey is a shared secret used for AES-256 encryption of all gossip traffic.
	// When set, both LAN and WAN pools require this key to join and communicate.
	// Must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256 respectively.
	SecretKey []byte

	// OnMessage is called when a broadcast message is received from the cluster.
	OnMessage func(msg *clusterv1.ClusterMessage)
}

func (c *Config) setDefaults() {
	if c.BindAddr == "" {
		c.BindAddr = "0.0.0.0"
	}
	// BindPort and WANBindPort default to 0, which lets the OS pick ephemeral
	// ports. In production, callers should set these explicitly (e.g. 7946/7947).
}
