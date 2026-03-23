package deployschedule

// Restate state keys for the DeploySchedulerService virtual object.
const (
	stateActiveSlots = "active_slots" // map[string]activeSlot
	stateWaitlist    = "waitlist"     // []waitlistEntry
)

// activeSlot tracks a build slot that has been acquired by a queue.
type activeSlot struct {
	AcquiredAt int64 `json:"acquired_at"`
}

// waitlistEntry represents a queue waiting for a build slot.
type waitlistEntry struct {
	QueueKey     string `json:"queue_key"`
	IsProduction bool   `json:"is_production"`
}
