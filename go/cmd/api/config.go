package api

type nodeConfig struct {
	Platform string `json:"platform,omitempty" description:"The platform this agent is running on"`
	NodeID   string `json:"nodeId,omitempty" description:"A unique node id"`
	Image    string `json:"image,omitempty" description:"The image this agent is running"`

	HttpAddr  string `json:"httpAddr" default:":7070" description:"Port to listen on"`
	RpcAddr   string `json:"rpcAddr" default:":7071" description:"The internal address for RPC"`
	Schema    string `json:"$schema,omitempty" description:"Make jsonschema happy"`
	Region    string `json:"region,omitempty" description:"The region this agent is running in"`
	Heartbeat *struct {
		URL      string `json:"url" minLength:"1" description:"URL to send heartbeat to"`
		Interval int    `json:"interval" min:"1" description:"Interval in seconds to send heartbeat"`
	} `json:"heartbeat,omitempty" description:"Send heartbeat to a URL"`

	Logs *struct {
		Color bool `json:"color" description:"Display color in logs"`
	} `json:"logs,omitempty"`
	RedisUrl   string `json:"redisUrl"`
	Clickhouse *struct {
		Url string `json:"url" minLength:"1"`
	} `json:"clickhouse,omitempty"`

	Database struct {
		// DSN of the primary database for reads and writes.
		Primary string `json:"primary"`

		// An optional read replica DSN.
		ReadonlyReplica string `json:"readonlyReplica,omitempty"`
	} `json:"database"`
}
