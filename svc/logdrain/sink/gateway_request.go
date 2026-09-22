package sink

// GatewayRequestPayload exports request metadata and details captured by the logging policy.
type GatewayRequestPayload struct {
	RequestID     string                `json:"request_id"`
	ProjectID     string                `json:"project_id"`
	AppID         string                `json:"app_id"`
	EnvironmentID string                `json:"environment_id"`
	DeploymentID  string                `json:"deployment_id"`
	Region        string                `json:"region"`
	Request       GatewayRequest        `json:"request"`
	Response      GatewayResponse       `json:"response"`
	Latency       GatewayRequestLatency `json:"latency"`
	// ErrorCode is the Unkey error code when the gateway rejected the request
	// itself, and empty when an instance served the response.
	ErrorCode string `json:"error_code"`
}

func (GatewayRequestPayload) isPayload() {}

type GatewayRequest struct {
	Method      string              `json:"method"`
	Host        string              `json:"host"`
	Path        string              `json:"path"`
	QueryString string              `json:"query_string"`
	QueryParams map[string][]string `json:"query_params"`
	Headers     []string            `json:"headers"`
	Body        string              `json:"body"`
	UserAgent   string              `json:"user_agent"`
	IPAddress   string              `json:"ip_address"`
}

type GatewayResponse struct {
	Status  int32    `json:"status"`
	Headers []string `json:"headers"`
	Body    string   `json:"body"`
}

type GatewayRequestLatency struct {
	Total    int64 `json:"total"`
	Instance int64 `json:"instance"`
	Gateway  int64 `json:"gateway"`
}
