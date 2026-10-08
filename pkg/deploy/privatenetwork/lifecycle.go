package privatenetwork

import "time"

// ReplacementOverlap is how long a replaced live production deployment stays on
// standby, and how long an obsolete discovery Service or connection grant stays
// available after it is scheduled for retirement. Replaced previews stop sooner.
const ReplacementOverlap = 30 * time.Minute

// RetireAfterAnnotation holds a discovery Service's retirement deadline as an
// RFC 3339 timestamp. Readers must reject the Service at or after the deadline.
const RetireAfterAnnotation = "private-network.unkey.com/retire-after"
