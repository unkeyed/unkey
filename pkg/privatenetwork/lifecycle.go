// Package privatenetwork holds the discovery contract shared by Krane, which
// publishes private network discovery objects, and undns, which reads them.
package privatenetwork

import "time"

// ReplacementOverlap is how long a replaced deployment and its discovery
// Service stay available after each is scheduled for retirement.
const ReplacementOverlap = 30 * time.Minute

// RetireAfterAnnotation holds a discovery Service's retirement deadline as an
// RFC 3339 timestamp. Readers must reject the Service at or after the deadline.
const RetireAfterAnnotation = "private-network.unkey.com/retire-after"
