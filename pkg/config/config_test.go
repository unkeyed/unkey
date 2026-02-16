package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadBytes_ParsesJSON(t *testing.T) {
	type cfg struct {
		Host string `json:"host" config:"required"`
		Port int    `json:"port" config:"default=3000"`
	}

	tests := []struct {
		name    string
		input   string
		wantErr string
		check   func(t *testing.T, c cfg)
	}{
		{
			name:  "all fields present are parsed",
			input: `{"host":"example.com","port":8080}`,
			check: func(t *testing.T, c cfg) {
				t.Helper()
				require.Equal(t, "example.com", c.Host)
				require.Equal(t, 8080, c.Port)
			},
		},
		{
			name:    "missing required field produces error",
			input:   `{"port":8080}`,
			wantErr: "Host",
		},
		{
			name:  "zero-value field receives default",
			input: `{"host":"example.com"}`,
			check: func(t *testing.T, c cfg) {
				t.Helper()
				require.Equal(t, 3000, c.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadBytes[cfg]([]byte(tt.input), JSON)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestLoadBytes_ParsesTOML(t *testing.T) {
	type cfg struct {
		Host string `toml:"host" config:"required"`
		Port int    `toml:"port" config:"default=3000"`
	}

	tests := []struct {
		name    string
		input   string
		wantErr string
		check   func(t *testing.T, c cfg)
	}{
		{
			name:  "all fields present are parsed",
			input: "host = \"example.com\"\nport = 8080\n",
			check: func(t *testing.T, c cfg) {
				t.Helper()
				require.Equal(t, "example.com", c.Host)
				require.Equal(t, 8080, c.Port)
			},
		},
		{
			name:    "missing required field produces error",
			input:   "port = 8080\n",
			wantErr: "Host",
		},
		{
			name:  "zero-value field receives default",
			input: "host = \"example.com\"\n",
			check: func(t *testing.T, c cfg) {
				t.Helper()
				require.Equal(t, 3000, c.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadBytes[cfg]([]byte(tt.input), TOML)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestLoadBytes_ParsesYAML(t *testing.T) {
	type cfg struct {
		Host string `yaml:"host" config:"required"`
		Port int    `yaml:"port" config:"default=3000"`
	}

	tests := []struct {
		name    string
		input   string
		wantErr string
		check   func(t *testing.T, c cfg)
	}{
		{
			name:  "all fields present are parsed",
			input: "host: example.com\nport: 8080\n",
			check: func(t *testing.T, c cfg) {
				t.Helper()
				require.Equal(t, "example.com", c.Host)
				require.Equal(t, 8080, c.Port)
			},
		},
		{
			name:    "missing required field produces error",
			input:   "port: 8080\n",
			wantErr: "Host",
		},
		{
			name:  "zero-value field receives default",
			input: "host: example.com\n",
			check: func(t *testing.T, c cfg) {
				t.Helper()
				require.Equal(t, 3000, c.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadBytes[cfg]([]byte(tt.input), YAML)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestLoadBytes_AppliesDefaults(t *testing.T) {
	t.Run("int field gets default", func(t *testing.T) {
		type cfg struct {
			Port int `yaml:"port" config:"default=8080"`
		}
		got, err := LoadBytes[cfg]([]byte("{}"), YAML)
		require.NoError(t, err)
		require.Equal(t, 8080, got.Port)
	})

	t.Run("string field gets default", func(t *testing.T) {
		type cfg struct {
			Host string `yaml:"host" config:"default=localhost"`
		}
		got, err := LoadBytes[cfg]([]byte("{}"), YAML)
		require.NoError(t, err)
		require.Equal(t, "localhost", got.Host)
	})

	t.Run("float field gets default", func(t *testing.T) {
		type cfg struct {
			Rate float64 `yaml:"rate" config:"default=0.5"`
		}
		got, err := LoadBytes[cfg]([]byte("{}"), YAML)
		require.NoError(t, err)
		require.InDelta(t, 0.5, got.Rate, 0.001)
	})

	t.Run("bool field gets default", func(t *testing.T) {
		type cfg struct {
			Debug bool `yaml:"debug" config:"default=true"`
		}
		got, err := LoadBytes[cfg]([]byte("{}"), YAML)
		require.NoError(t, err)
		require.True(t, got.Debug)
	})

	t.Run("duration field gets default", func(t *testing.T) {
		type cfg struct {
			Timeout time.Duration `yaml:"timeout" config:"default=5s"`
		}
		got, err := LoadBytes[cfg]([]byte("{}"), YAML)
		require.NoError(t, err)
		require.Equal(t, 5*time.Second, got.Timeout)
	})

	t.Run("explicit value is not overwritten by default", func(t *testing.T) {
		type cfg struct {
			Port int `yaml:"port" config:"default=8080"`
		}
		got, err := LoadBytes[cfg]([]byte("port: 9090"), YAML)
		require.NoError(t, err)
		require.Equal(t, 9090, got.Port)
	})
}

func TestLoadBytes_ValidatesRequired(t *testing.T) {
	t.Run("empty string fails required", func(t *testing.T) {
		type cfg struct {
			Name string `yaml:"name" config:"required"`
		}
		_, err := LoadBytes[cfg]([]byte("{}"), YAML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Name")
	})

	t.Run("set string passes required", func(t *testing.T) {
		type cfg struct {
			Name string `yaml:"name" config:"required"`
		}
		_, err := LoadBytes[cfg]([]byte("name: hello"), YAML)
		require.NoError(t, err)
	})

	t.Run("nil slice fails required", func(t *testing.T) {
		type cfg struct {
			Items []string `yaml:"items" config:"required"`
		}
		_, err := LoadBytes[cfg]([]byte("{}"), YAML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Items")
	})
}

func TestLoadBytes_ValidatesNumericBounds(t *testing.T) {
	type minCfg struct {
		Count int `yaml:"count" config:"min=1"`
	}
	type maxCfg struct {
		Count int `yaml:"count" config:"max=100"`
	}

	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{
			name: "value below min is rejected",
			run: func() error {
				_, err := LoadBytes[minCfg]([]byte("count: 0"), YAML)
				return err
			},
			wantErr: true,
		},
		{
			name: "value at min is accepted",
			run: func() error {
				_, err := LoadBytes[minCfg]([]byte("count: 1"), YAML)
				return err
			},
		},
		{
			name: "value above min is accepted",
			run: func() error {
				_, err := LoadBytes[minCfg]([]byte("count: 5"), YAML)
				return err
			},
		},
		{
			name: "value above max is rejected",
			run: func() error {
				_, err := LoadBytes[maxCfg]([]byte("count: 101"), YAML)
				return err
			},
			wantErr: true,
		},
		{
			name: "value at max is accepted",
			run: func() error {
				_, err := LoadBytes[maxCfg]([]byte("count: 100"), YAML)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "Count")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadBytes_ValidatesStringLength(t *testing.T) {
	t.Run("string shorter than minLength is rejected", func(t *testing.T) {
		type cfg struct {
			Code string `yaml:"code" config:"minLength=3"`
		}
		_, err := LoadBytes[cfg]([]byte("code: ab"), YAML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Code")
	})

	t.Run("string at minLength is accepted", func(t *testing.T) {
		type cfg struct {
			Code string `yaml:"code" config:"minLength=3"`
		}
		_, err := LoadBytes[cfg]([]byte("code: abc"), YAML)
		require.NoError(t, err)
	})

	t.Run("string longer than maxLength is rejected", func(t *testing.T) {
		type cfg struct {
			Code string `yaml:"code" config:"maxLength=5"`
		}
		_, err := LoadBytes[cfg]([]byte("code: abcdef"), YAML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Code")
	})

	t.Run("string at maxLength is accepted", func(t *testing.T) {
		type cfg struct {
			Code string `yaml:"code" config:"maxLength=5"`
		}
		_, err := LoadBytes[cfg]([]byte("code: abcde"), YAML)
		require.NoError(t, err)
	})
}

func TestLoadBytes_ValidatesNonempty(t *testing.T) {
	t.Run("empty string is rejected", func(t *testing.T) {
		type cfg struct {
			Name string `yaml:"name" config:"nonempty"`
		}
		_, err := LoadBytes[cfg]([]byte(`name: ""`), YAML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Name")
	})

	t.Run("empty slice is rejected", func(t *testing.T) {
		type cfg struct {
			Items []string `yaml:"items" config:"nonempty"`
		}
		_, err := LoadBytes[cfg]([]byte("items: []"), YAML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Items")
	})

	t.Run("non-empty slice is accepted", func(t *testing.T) {
		type cfg struct {
			Items []string `yaml:"items" config:"nonempty"`
		}
		_, err := LoadBytes[cfg]([]byte("items:\n  - a"), YAML)
		require.NoError(t, err)
	})
}

func TestLoadBytes_ValidatesOneof(t *testing.T) {
	type cfg struct {
		Mode string `yaml:"mode" config:"oneof=a|b|c"`
	}

	t.Run("value in set is accepted", func(t *testing.T) {
		_, err := LoadBytes[cfg]([]byte("mode: b"), YAML)
		require.NoError(t, err)
	})

	t.Run("value not in set is rejected", func(t *testing.T) {
		_, err := LoadBytes[cfg]([]byte("mode: d"), YAML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Mode")
	})
}

func TestLoadBytes_CollectsAllValidationErrors(t *testing.T) {
	type cfg struct {
		A string `yaml:"a" config:"required"`
		B string `yaml:"b" config:"required"`
		C string `yaml:"c" config:"required"`
	}

	_, err := LoadBytes[cfg]([]byte("{}"), YAML)
	require.Error(t, err)

	msg := err.Error()
	require.Contains(t, msg, "A")
	require.Contains(t, msg, "B")
	require.Contains(t, msg, "C")
}

func TestLoadBytes_ValidatesNestedStructFields(t *testing.T) {
	type dbCfg struct {
		Primary string `yaml:"primary" config:"required"`
	}
	type cfg struct {
		Database dbCfg `yaml:"database"`
	}

	_, err := LoadBytes[cfg]([]byte("database:\n  primary: \"\""), YAML)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Database.Primary")
}

func TestLoadBytes_CallsValidatorInterface(t *testing.T) {
	_, err := LoadBytes[validatedCfg]([]byte("port: 0"), YAML)
	require.Error(t, err)
	require.Contains(t, err.Error(), "port must be positive")
}

// validatedCfg is defined at file scope so the Validate method can be declared.
type validatedCfg struct {
	Port int `yaml:"port"`
}

func (c *validatedCfg) Validate() error {
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	return nil
}

func TestLoadBytes_ExpandsEnvVars(t *testing.T) {
	type cfg struct {
		Secret string `yaml:"secret"`
	}

	t.Setenv("CONFIG_TEST_SECRET", "hunter2")

	got, err := LoadBytes[cfg]([]byte("secret: ${CONFIG_TEST_SECRET}"), YAML)
	require.NoError(t, err)
	require.Equal(t, "hunter2", got.Secret)
}

func TestLoadBytes_ExpandsEnvVarsWithDefault(t *testing.T) {
	type cfg struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	}

	t.Run("uses default when env var is unset", func(t *testing.T) {
		got, err := LoadBytes[cfg]([]byte("host: ${CONFIG_TEST_UNSET_VAR:-localhost}"), YAML)
		require.NoError(t, err)
		require.Equal(t, "localhost", got.Host)
	})

	t.Run("uses default when env var is empty", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_EMPTY_VAR", "")

		got, err := LoadBytes[cfg]([]byte("host: ${CONFIG_TEST_EMPTY_VAR:-fallback}"), YAML)
		require.NoError(t, err)
		require.Equal(t, "fallback", got.Host)
	})

	t.Run("uses env value when set", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_SET_VAR", "prod.example.com")

		got, err := LoadBytes[cfg]([]byte("host: ${CONFIG_TEST_SET_VAR:-localhost}"), YAML)
		require.NoError(t, err)
		require.Equal(t, "prod.example.com", got.Host)
	})

	t.Run("empty default is valid", func(t *testing.T) {
		got, err := LoadBytes[cfg]([]byte("host: ${CONFIG_TEST_UNSET_VAR2:-}"), YAML)
		require.NoError(t, err)
		require.Equal(t, "", got.Host)
	})
}

func TestLoad_DetectsFormatFromExtension(t *testing.T) {
	type cfg struct {
		Host string `json:"host" yaml:"host"`
	}

	t.Run("yaml extension", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		require.NoError(t, os.WriteFile(path, []byte("host: example.com"), 0o644))

		got, err := Load[cfg](path)
		require.NoError(t, err)
		require.Equal(t, "example.com", got.Host)
	})

	t.Run("json extension", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"host":"example.com"}`), 0o644))

		got, err := Load[cfg](path)
		require.NoError(t, err)
		require.Equal(t, "example.com", got.Host)
	})

	t.Run("toml extension", func(t *testing.T) {
		type tomlCfg struct {
			Host string `toml:"host"`
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		require.NoError(t, os.WriteFile(path, []byte("host = \"example.com\""), 0o644))

		got, err := Load[tomlCfg](path)
		require.NoError(t, err)
		require.Equal(t, "example.com", got.Host)
	})

	t.Run("unsupported extension returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.txt")
		require.NoError(t, os.WriteFile(path, []byte("host: example.com"), 0o644))

		_, err := Load[cfg](path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported")
	})
}

func TestSchema_MapsGoTypesToJSONSchemaTypes(t *testing.T) {
	type basic struct {
		Host    string   `json:"host"`
		Port    int      `json:"port"`
		Rate    float64  `json:"rate"`
		Debug   bool     `json:"debug"`
		Brokers []string `json:"brokers"`
	}

	data, err := Schema[basic]()
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"])
	require.Equal(t, "object", schema["type"])
	require.Equal(t, false, schema["additionalProperties"])

	props := schema["properties"].(map[string]any)
	require.Equal(t, "string", props["host"].(map[string]any)["type"])
	require.Equal(t, "integer", props["port"].(map[string]any)["type"])
	require.Equal(t, "number", props["rate"].(map[string]any)["type"])
	require.Equal(t, "boolean", props["debug"].(map[string]any)["type"])
	require.Equal(t, "array", props["brokers"].(map[string]any)["type"])
}

func TestSchema_MapsConfigTagsToJSONSchemaKeywords(t *testing.T) {
	type schemaConfig struct {
		Region  string   `json:"region"  config:"required,oneof=aws|gcp"`
		Port    int      `json:"port"    config:"default=8080,min=1,max=65535"`
		Name    string   `json:"name"    config:"minLength=1,maxLength=100"`
		Brokers []string `json:"brokers" config:"nonempty"`
	}

	data, err := Schema[schemaConfig]()
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	requiredArr := schema["required"].([]any)
	require.Contains(t, requiredArr, "region")

	props := schema["properties"].(map[string]any)

	regionProp := props["region"].(map[string]any)
	require.Equal(t, []any{"aws", "gcp"}, regionProp["enum"])

	portProp := props["port"].(map[string]any)
	require.Equal(t, float64(8080), portProp["default"])
	require.Equal(t, float64(1), portProp["minimum"])
	require.Equal(t, float64(65535), portProp["maximum"])

	nameProp := props["name"].(map[string]any)
	require.Equal(t, float64(1), nameProp["minLength"])
	require.Equal(t, float64(100), nameProp["maxLength"])

	brokersProp := props["brokers"].(map[string]any)
	require.Equal(t, float64(1), brokersProp["minItems"])
}

func TestSchema_ProducesNestedObjects(t *testing.T) {
	type inner struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	type outer struct {
		Database inner `json:"database"`
	}

	data, err := Schema[outer]()
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	props := schema["properties"].(map[string]any)
	dbProp := props["database"].(map[string]any)
	require.Equal(t, "object", dbProp["type"])

	dbProps := dbProp["properties"].(map[string]any)
	require.Equal(t, "string", dbProps["host"].(map[string]any)["type"])
	require.Equal(t, "integer", dbProps["port"].(map[string]any)["type"])
}

func TestSchema_FallsBackToYamlTagForFieldNames(t *testing.T) {
	type yamlOnly struct {
		Listen string `yaml:"listen_addr"`
		Count  int    `yaml:"count"`
	}

	data, err := Schema[yamlOnly]()
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	props := schema["properties"].(map[string]any)
	require.Contains(t, props, "listen_addr")
	require.Contains(t, props, "count")
}

func TestSchema_FallsBackToTomlTagForFieldNames(t *testing.T) {
	type tomlOnly struct {
		Listen string `toml:"listen_addr"`
		Count  int    `toml:"count"`
	}

	data, err := Schema[tomlOnly]()
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	props := schema["properties"].(map[string]any)
	require.Contains(t, props, "listen_addr")
	require.Contains(t, props, "count")
}
