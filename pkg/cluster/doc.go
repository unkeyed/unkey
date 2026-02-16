// Package cluster provides a two-tier gossip-based cluster membership using
// hashicorp/memberlist (SWIM protocol).
//
// Architecture:
//
//   - LAN pool: all nodes in a region, using DefaultLANConfig (~1ms propagation)
//   - WAN pool: one ambassador per region (auto-elected oldest node), DefaultWANConfig
//
// Message flow for cache invalidation:
//
//	node → LAN broadcast → ambassador → WAN → remote ambassadors → their LAN pools
//
// Ambassador election: the oldest node in the LAN pool (by join time encoded in
// the memberlist node name) automatically becomes the WAN ambassador. When the
// ambassador leaves, the next oldest node promotes itself.
package cluster
