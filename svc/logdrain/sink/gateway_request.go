package sink

// GatewayRequestPayload exports customer request metadata, without captured secrets or internal addresses.
type GatewayRequestPayload struct {
	RequestID       string `json:"request_id"`
	ProjectID       string `json:"project_id"`
	AppID           string `json:"app_id"`
	EnvironmentID   string `json:"environment_id"`
	DeploymentID    string `json:"deployment_id"`
	Region          string `json:"region"`
	Method          string `json:"method"`
	Host            string `json:"host"`
	Path            string `json:"path"`
	ResponseStatus  int32  `json:"response_status"`
	TotalLatency    int64  `json:"total_latency"`
	InstanceLatency int64  `json:"instance_latency"`
	GatewayLatency  int64  `json:"gateway_latency"`
}

func (GatewayRequestPayload) isPayload() {}
