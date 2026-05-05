package bus

import "errors"

// Config configures a Serf-backed Bus.
type Config struct {
	// Region is the geographic region this node runs in (e.g. "us-east-1").
	// Used as the BusEnvelope.SourceRegion on every publish and as the
	// "region" tag advertised to peers.
	Region string

	// NodeID is a globally-unique identifier for this process. Used as the
	// Serf node name and the dedup key partition.
	NodeID string

	// BindAddr is the local interface to bind on. Default "0.0.0.0".
	BindAddr string

	// BindPort is the gossip port (TCP+UDP). Default 7946.
	BindPort int

	// AdvertiseAddr is the address peers should reach this node on. For
	// pods inside a peered VPC, this is the pod IP; the cross-region NLB is
	// for inbound discovery only and never advertised. Empty leaves Serf to
	// pick the bind address.
	AdvertiseAddr string

	// Seeds are addresses to dial at startup. The first reachable one is
	// enough to bootstrap. Mix the local headless service with the
	// cross-region NLB hostnames; order does not matter.
	Seeds []string

	// SecretKey is a 16/24/32-byte AES key for encrypting all gossip
	// traffic. Required in any environment where the gossip port is
	// reachable from outside a tightly-controlled network boundary.
	SecretKey []byte

	// Tags are advertised to every peer (role, region, version, instance).
	// Subscribers can filter on tags via Serf queries; the API service
	// reads them through Bus.Members.
	Tags map[string]string

	// MaxUserEventSize bounds the marshalled BusEnvelope size. Default 512
	// matches Serf's UserEventSizeLimit; raise it if a topic ships larger
	// payloads (and pay the cross-region bandwidth cost).
	MaxUserEventSize int

	// DedupCacheSize is the maximum number of (sender_node, id) pairs
	// remembered to drop duplicates. Default 16384. Sized at 16k assuming
	// ~10 events/s sustained per pod; ~25 minutes of memory at that rate.
	DedupCacheSize int

	// ReplayLogBytesPerTopic caps the size of a single topic's ring buffer.
	// Default 4 MiB. A single noisy topic cannot evict events from quieter
	// topics until both this and ReplayLogBytesTotal are exceeded.
	ReplayLogBytesPerTopic int

	// ReplayLogBytesTotal caps the aggregate size across all topic rings on
	// this pod. Default 16 MiB. When hit, the oldest entry in the largest
	// ring is evicted first.
	ReplayLogBytesTotal int
}

func (c *Config) setDefaults() {
	if c.BindAddr == "" {
		c.BindAddr = "0.0.0.0"
	}
	if c.BindPort == 0 {
		c.BindPort = 7946
	}
	if c.MaxUserEventSize == 0 {
		c.MaxUserEventSize = 512
	}
	if c.DedupCacheSize == 0 {
		c.DedupCacheSize = 16384
	}
	if c.ReplayLogBytesPerTopic == 0 {
		c.ReplayLogBytesPerTopic = 4 << 20
	}
	if c.ReplayLogBytesTotal == 0 {
		c.ReplayLogBytesTotal = 16 << 20
	}
}

func (c *Config) validate() error {
	if c.NodeID == "" {
		return errors.New("bus: NodeID is required")
	}
	if c.Region == "" {
		return errors.New("bus: Region is required")
	}
	if len(c.SecretKey) > 0 {
		switch len(c.SecretKey) {
		case 16, 24, 32:
		default:
			return errors.New("bus: SecretKey must be 16, 24, or 32 bytes")
		}
	}
	return nil
}
