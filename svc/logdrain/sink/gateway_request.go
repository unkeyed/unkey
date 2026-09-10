package sink

// GatewayRequestPayload exports request metadata and details captured by the logging policy.
type GatewayRequestPayload struct {
	RequestID       string              `json:"request_id"`
	ProjectID       string              `json:"project_id"`
	AppID           string              `json:"app_id"`
	EnvironmentID   string              `json:"environment_id"`
	DeploymentID    string              `json:"deployment_id"`
	Region          string              `json:"region"`
	Method          string              `json:"method"`
	Host            string              `json:"host"`
	Path            string              `json:"path"`
	QueryString     string              `json:"query_string"`
	QueryParams     map[string][]string `json:"query_params"`
	RequestHeaders  []string            `json:"request_headers"`
	RequestBody     string              `json:"request_body"`
	ResponseHeaders []string            `json:"response_headers"`
	ResponseBody    string              `json:"response_body"`
	UserAgent       string              `json:"user_agent"`
	IPAddress       string              `json:"ip_address"`
	ResponseStatus  int32               `json:"response_status"`
	TotalLatency    int64               `json:"total_latency"`
	InstanceLatency int64               `json:"instance_latency"`
	GatewayLatency  int64               `json:"gateway_latency"`
}

func (GatewayRequestPayload) isPayload() {}
