package deployqueue

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// Restate state keys for the DeployQueueService virtual object.
const (
	stateQueue       = "queue"        // []queueEntry
	stateActive      = "active"       // *activeDeploy
	stateWorkspaceID = "workspace_id" // string
)

// queueEntry represents a pending deploy request in the queue.
// DeployReqBytes stores the DeployRequest as protojson bytes because Restate
// state uses encoding/json which can't round-trip protobuf oneof fields.
type queueEntry struct {
	DeploymentID   string `json:"deployment_id"`
	DeployReqBytes []byte `json:"deploy_request_bytes"`
	IsProduction   bool   `json:"is_production"`
	Branch         string `json:"branch"`
	EnqueuedAt     int64  `json:"enqueued_at"`
}

// marshalDeployRequest serializes a DeployRequest to protojson bytes for storage.
func marshalDeployRequest(req *hydrav1.DeployRequest) ([]byte, error) {
	return protojson.Marshal(req)
}

// unmarshalDeployRequest deserializes a DeployRequest from protojson bytes.
func unmarshalDeployRequest(b []byte) (*hydrav1.DeployRequest, error) {
	req := &hydrav1.DeployRequest{}
	if err := protojson.Unmarshal(b, req); err != nil {
		return nil, err
	}
	return req, nil
}

// activeDeploy tracks the currently executing deploy for this queue.
type activeDeploy struct {
	DeploymentID string `json:"deployment_id"`
	InvocationID string `json:"invocation_id"`
	IsProduction bool   `json:"is_production"`
}
