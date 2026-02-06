package cluster

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

	// OnMessage is called when a broadcast message is received from the cluster.
	// The payload is the raw application bytes (no framing).
	OnMessage func(msg []byte)
}

func (c *Config) setDefaults() {
	if c.BindAddr == "" {
		c.BindAddr = "0.0.0.0"
	}
	// BindPort and WANBindPort default to 0, which lets the OS pick ephemeral
	// ports. In production, callers should set these explicitly (e.g. 7946/7947).
}
