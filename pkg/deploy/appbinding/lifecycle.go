// Package appbinding defines the app binding discovery contract shared by
// Ctrl, Krane, and undns.
package appbinding

import "time"

// ReplacementOverlap is how long a replaced deployment and its discovery
// Service stay available after each is scheduled for retirement.
const ReplacementOverlap = 30 * time.Minute

// RetireAfterAnnotation holds a discovery Service's retirement deadline as an
// RFC 3339 timestamp. Readers must reject the Service at or after the deadline.
const RetireAfterAnnotation = "private-network.unkey.com/retire-after"
