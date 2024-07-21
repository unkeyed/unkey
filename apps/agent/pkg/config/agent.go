package config

type Agent struct {
	Platform string `json:"platform,omitempty" description:"The platform this agent is running on"`
	NodeId   string `json:"nodeId,omitempty" description:"A unique node id"`
	Image    string `json:"image,omitempty" description:"The image this agent is running"`
	Pprof    *struct {
		Username string `json:"username,omitempty" description:"The username to use for pprof"`
		Password string `json:"password,omitempty" description:"The password to use for pprof"`
	} `json:"pprof,omitempty" description:"Enable pprof"`
	Logging *struct {
		Axiom *struct {
			Dataset string `json:"dataset" minLength:"1" description:"The dataset to send logs to"`
			Token   string `json:"token" minLength:"1" description:"The token to use for authentication"`
		} `json:"axiom,omitempty" description:"Send logs to axiom"`
	} `json:"logging,omitempty"`

	Tracing *struct {
		Axiom *struct {
			Dataset string `json:"dataset" minLength:"1" description:"The dataset to send traces to"`
			Token   string `json:"token" minLength:"1" description:"The token to use for authentication"`
		} `json:"axiom,omitempty" description:"Send traces to axiom"`
	} `json:"tracing,omitempty"`

	Metrics *struct {
		Axiom *struct {
			Dataset string `json:"dataset" minLength:"1" description:"The dataset to send metrics to"`
			Token   string `json:"token" minLength:"1" description:"The token to use for authentication"`
		} `json:"axiom,omitempty" description:"Send metrics to axiom"`
	} `json:"metrics,omitempty"`

	Schema    string `json:"$schema,omitempty" description:"Make jsonschema happy"`
	Region    string `json:"region,omitempty" description:"The region this agent is running in"`
	Port      string `json:"port,omitempty" default:"8080" description:"Port to listen on"`
	Heartbeat *struct {
		URL      string `json:"url" minLength:"1" description:"URL to send heartbeat to"`
		Interval int    `json:"interval" min:"1" description:"Interval in seconds to send heartbeat"`
	} `json:"heartbeat,omitempty" description:"Send heartbeat to a URL"`

	Services struct {
		Ratelimit *struct {
			AuthToken string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
		} `json:"ratelimit,omitempty" description:"Rate limit requests"`
		EventRouter *struct {
			AuthToken string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
			Tinybird  *struct {
				Token         string `json:"token" minLength:"1" description:"The token to use for tinybird authentication"`
				FlushInterval int    `json:"flushInterval" min:"1" description:"Interval in seconds to flush events"`
				BufferSize    int    `json:"bufferSize" min:"1" description:"Size of the buffer"`
				BatchSize     int    `json:"batchSize" min:"1" description:"Size of the batch"`
			} `json:"tinybird,omitempty" description:"Send events to tinybird"`
		} `json:"eventRouter,omitempty" description:"Route events"`
		Vault *struct {
			S3Bucket          string `json:"s3Bucket" minLength:"1" description:"The bucket to store secrets in"`
			S3Url             string `json:"s3Url" minLength:"1" description:"The url to store secrets in"`
			S3AccessKeyId     string `json:"s3AccessKeyId" minLength:"1" description:"The access key id to use for s3"`
			S3AccessKeySecret string `json:"s3AccessKeySecret" minLength:"1" description:"The access key secret to use for s3"`
			MasterKeys        string `json:"masterKeys" minLength:"1" description:"The master keys to use for encryption, comma separated"`
			AuthToken         string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
		} `json:"vault,omitempty" description:"Store secrets"`
	} `json:"services"`

	Cluster *struct {
		AuthToken string `json:"authToken" minLength:"1" description:"The token to use for http authentication"`
		SerfAddr  string `json:"serfAddr" minLength:"1" description:"The host and port for serf to listen on"`
		RpcAddr   string `json:"rpcAddr" minLength:"1" description:"This node's internal address, including protocol and port"`

		Join *struct {
			Env *struct {
				Addrs []string `json:"addrs" description:"Addresses to join, comma separated"`
			} `json:"env,omitempty"`
			Dns *struct {
				AAAA string `json:"aaaa" description:"The AAAA record that returns a comma separated list, containing the ipv6 addresses of all nodes"`
			} `json:"dns,omitempty"`
		} `json:"join,omitempty" description:"The strategy to use to join the cluster"`
	} `json:"cluster,omitempty"`

	Pyroscope *struct {
		Url      string `json:"url" minLength:"1"`
		User     string `json:"user" minLength:"1"`
		Password string `json:"password" minLength:"1"`
	} `json:"pyroscope,omitempty"`
}
