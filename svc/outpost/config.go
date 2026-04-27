package outpost

import (
	"fmt"
)

type CAConfig struct {
	CertFile string `toml:"cert_file" config:"required,nonempty"`
	KeyFile  string `toml:"key_file" config:"required,nonempty"`
}

type ClickHouseConfig struct {
	URL        string `toml:"url"`
	BatchSize  int    `toml:"batch_size" config:"default=5000,min=1"`
	BufferSize int    `toml:"buffer_size" config:"default=10000,min=1"`
	Consumers  int    `toml:"consumers" config:"default=1,min=1"`
}

type Config struct {
	InstanceID string           `toml:"instance_id"`
	ProxyPort  int              `toml:"proxy_port" config:"default=3128,min=1,max=65535"`
	HttpPort   int              `toml:"http_port" config:"default=8090,min=1,max=65535"`
	Region     string           `toml:"region" config:"required,nonempty"`
	Platform   string           `toml:"platform" config:"required,nonempty"`
	CA         CAConfig         `toml:"ca"`
	ClickHouse ClickHouseConfig `toml:"clickhouse"`
}

func (c *Config) Validate() error {
	if c.ProxyPort == c.HttpPort {
		return fmt.Errorf("proxy_port and http_port must be different")
	}
	return nil
}
