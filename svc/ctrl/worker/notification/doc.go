// Package notification routes customer events to frozen, encrypted delivery targets.
// Register event definitions and channel providers with [New], then bind both
// [Service.Definition] and [Service.DeliveryDefinition] to the Restate server.
//
// Routing is frozen once per event identity. Each delivery rechecks current member
// eligibility or destination state before sending. Provider acceptance does not
// confirm receipt; ambiguous results are retained rather than blindly retried.
// [Dispatcher] also routes events to stateful integrations outside channel delivery.
package notification
