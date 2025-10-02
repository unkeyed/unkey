package kafka

// Config contains Kafka-specific configuration for analytics.
type Config struct {
	// Brokers is a list of Kafka broker addresses
	Brokers []string `json:"brokers"`

	// Topics configuration for different event types
	Topics Topics `json:"topics"`

	// ProducerConfig contains additional producer settings
	ProducerConfig map[string]interface{} `json:"producer_config,omitempty"`
}

// Topics defines the topic names for different event types.
type Topics struct {
	// KeyVerifications topic for key verification events
	KeyVerifications string `json:"key_verifications"`

	// Ratelimits topic for ratelimit events
	Ratelimits string `json:"ratelimits"`

	// ApiRequests topic for API request events
	ApiRequests string `json:"api_requests"`
}

// DefaultTopics returns the default topic configuration.
func DefaultTopics() Topics {
	return Topics{
		KeyVerifications: "analytics.key_verifications",
		Ratelimits:       "analytics.ratelimits",
		ApiRequests:      "analytics.api_requests",
	}
}

// DefaultConfig returns a default Kafka configuration.
func DefaultConfig(brokers []string) Config {
	return Config{
		Brokers: brokers,
		Topics:  DefaultTopics(),
	}
}
