package testharness

import (
	"sync"
	"time"
)

// WorkflowContext interface for extracting metadata (avoid import cycle)
type WorkflowContext interface {
	ExecutionID() string
	WorkflowName() string
}

// EventType represents the type of event that occurred
type EventType string

const (
	WorkflowStarted   EventType = "workflow_started"
	WorkflowCompleted EventType = "workflow_completed"
	WorkflowFailed    EventType = "workflow_failed"
	StepExecuting     EventType = "step_executing"
	StepExecuted      EventType = "step_executed"
	StepFailed        EventType = "step_failed"
)

// EventRecord represents something that happened during test execution
type EventRecord struct {
	Type      EventType              `json:"type"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// EventCollector captures events during test execution
type EventCollector struct {
	mu     sync.RWMutex
	events []EventRecord
}

// NewEventCollector creates a new event collector
func NewEventCollector() *EventCollector {
	return &EventCollector{
		mu:     sync.RWMutex{},
		events: make([]EventRecord, 0),
	}
}

// Emit records an event with workflow context metadata automatically included
func (e *EventCollector) Emit(ctx WorkflowContext, eventType EventType, message string, extraData ...interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Start with context metadata
	data := map[string]interface{}{
		"execution_id":  ctx.ExecutionID(),
		"workflow_name": ctx.WorkflowName(),
	}

	// Add extra data as key-value pairs
	for i := 0; i < len(extraData); i += 2 {
		if i+1 < len(extraData) {
			if key, ok := extraData[i].(string); ok {
				data[key] = extraData[i+1]
			}
		}
	}

	event := EventRecord{
		Type:      eventType,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data,
	}

	e.events = append(e.events, event)
}

// Events returns all collected events
func (e *EventCollector) Events() []EventRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to prevent race conditions
	events := make([]EventRecord, len(e.events))
	copy(events, e.events)
	return events
}

// Filter returns events that match the given criteria
func (e *EventCollector) Filter(eventType EventType) []EventRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var filtered []EventRecord
	for _, event := range e.events {
		if event.Type == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// FilterWithData returns events that match the type and have specific data values
func (e *EventCollector) FilterWithData(eventType EventType, key string, value interface{}) []EventRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var filtered []EventRecord
	for _, event := range e.events {
		if event.Type == eventType {
			if eventValue, exists := event.Data[key]; exists && eventValue == value {
				filtered = append(filtered, event)
			}
		}
	}
	return filtered
}

// Count returns the number of events of a specific type
func (e *EventCollector) Count(eventType EventType) int {
	return len(e.Filter(eventType))
}

// CountWithData returns the number of events that match type and data criteria
func (e *EventCollector) CountWithData(eventType EventType, key string, value interface{}) int {
	return len(e.FilterWithData(eventType, key, value))
}

// Clear removes all collected events
func (e *EventCollector) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = e.events[:0]
}

// GetLatest returns the most recent event of a given type, or nil if none found
func (e *EventCollector) GetLatest(eventType EventType) *EventRecord {
	events := e.Filter(eventType)
	if len(events) == 0 {
		return nil
	}
	return &events[len(events)-1]
}

// GetFirst returns the first event of a given type, or nil if none found
func (e *EventCollector) GetFirst(eventType EventType) *EventRecord {
	events := e.Filter(eventType)
	if len(events) == 0 {
		return nil
	}
	return &events[0]
}

// EventsBetween returns events that occurred between start and end times (inclusive)
func (e *EventCollector) EventsBetween(start, end time.Time) []EventRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var filtered []EventRecord
	for _, event := range e.events {
		if (event.Timestamp.Equal(start) || event.Timestamp.After(start)) &&
			(event.Timestamp.Equal(end) || event.Timestamp.Before(end)) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// Summary returns a summary of all event types and their counts
func (e *EventCollector) Summary() map[string]int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	summary := make(map[string]int)
	for _, event := range e.events {
		summary[string(event.Type)]++
	}
	return summary
}
