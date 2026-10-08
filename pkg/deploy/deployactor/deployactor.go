// Package deployactor holds the actor that svc/ctrl records on deployments
// Unkey starts itself. The API read path matches OpsID to report the actor as
// the system
package deployactor

const (
	// OpsID is the actor id of every operator-started rebuild. The ops bearer
	// token is the only identity at that boundary, so all rebuilds share it
	OpsID   = "unkey-ops"
	OpsName = "Unkey Ops"
)
