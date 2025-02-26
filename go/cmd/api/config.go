package api

type nodeConfig struct {
	Platform  string `json:"platform,omitempty" description:"Operating system platform identifier (e.g., linux, darwin, windows)"`
	Image     string `json:"image,omitempty" description:"Container image identifier including repository and tag"`
	HttpPort  int    `json:"httpPort" default:"7070" description:"HTTP port for the API server to listen on"`
	Schema    string `json:"$schema,omitempty" description:"JSON Schema URI for configuration validation"`
	Region    string `json:"region,omitempty" description:"Geographic region identifier where this node is deployed"`
	Heartbeat *struct {
		URL      string `json:"url" minLength:"1" description:"Complete URL endpoint where heartbeat signals will be sent"`
		Interval int    `json:"interval" min:"1" description:"Time between heartbeat signals in seconds"`
	} `json:"heartbeat,omitempty" description:"Configuration for health check heartbeat mechanism"`

	Cluster *struct {
		NodeID        string `json:"nodeId,omitempty" description:"Unique identifier for this node within the cluster"`
		AdvertiseAddr struct {
			Static         *string `json:"static,omitempty" description:"Static IP address or hostname for node discovery"`
			AwsEcsMetadata *bool   `json:"awsEcsMetadata,omitempty" description:"Enable automatic address discovery using AWS ECS container metadata"`
		} `json:"advertiseAddr" description:"Node address advertisement configuration for cluster communication"`
		RpcPort    string `json:"rpcPort" default:"7071" description:"Port used for internal RPC communication between nodes"`
		GossipPort string `json:"gossipPort" default:"7072" description:"Port used for cluster membership and failure detection"`
		Discovery  *struct {
			Static *struct {
				Addrs []string `json:"addrs" minLength:"1" description:"List of seed node addresses for static cluster configuration"`
			} `json:"static,omitempty" description:"Static cluster membership configuration"`
			Redis *struct {
				URL string `json:"url" minLength:"1" description:"Redis connection string for dynamic cluster discovery"`
			} `json:"redis,omitempty" description:"Redis-based cluster discovery configuration"`
		} `json:"discovery,omitempty" description:"Cluster node discovery mechanism configuration"`
	} `json:"cluster,omitempty" description:"Distributed cluster configuration settings"`

	Logs *struct {
		Color bool `json:"color" description:"Enable ANSI color codes in log output"`
	} `json:"logs,omitempty"`
	Clickhouse *struct {
		Url string `json:"url" minLength:"1" description:"ClickHouse database connection string"`
	} `json:"clickhouse,omitempty"`

	Database struct {
		Primary         string `json:"primary" description:"Primary database connection string for read and write operations"`
		ReadonlyReplica string `json:"readonlyReplica,omitempty" description:"Optional read-replica database connection string for read operations"`
	} `json:"database"`

	Otel *struct {
		OtlpEndpoint string `json:"otlpEndpoint" description:"OpenTelemetry collector endpoint for metrics, traces, and logs"`
	} `json:"otel,omitempty" description:"OpenTelemetry observability configuration"`
}
