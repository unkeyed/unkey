package sink

import "encoding/json"

// RuntimeLogPayload exports customer application output, without infrastructure metadata.
type RuntimeLogPayload struct {
	LogID         string          `json:"log_id"`
	Severity      string          `json:"severity"`
	Message       string          `json:"message"`
	Attributes    json.RawMessage `json:"attributes"`
	ProjectID     string          `json:"project_id"`
	AppID         string          `json:"app_id"`
	EnvironmentID string          `json:"environment_id"`
	DeploymentID  string          `json:"deployment_id"`
	Region        string          `json:"region"`
}

func (RuntimeLogPayload) isPayload() {}
