package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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

func TestLoadBytes_AppliesDefaults(t *testing.T) {
	t.Run("int field gets default", func(t *testing.T) {
		type cfg struct {
			Port int `toml:"port" config:"default=8080"`
		}
		got, err := LoadBytes[cfg]([]byte(""), TOML)
		require.NoError(t, err)
		require.Equal(t, 8080, got.Port)
	})

	t.Run("string field gets default", func(t *testing.T) {
		type cfg struct {
			Host string `toml:"host" config:"default=localhost"`
		}
		got, err := LoadBytes[cfg]([]byte(""), TOML)
		require.NoError(t, err)
		require.Equal(t, "localhost", got.Host)
	})

	t.Run("float field gets default", func(t *testing.T) {
		type cfg struct {
			Rate float64 `toml:"rate" config:"default=0.5"`
		}
		got, err := LoadBytes[cfg]([]byte(""), TOML)
		require.NoError(t, err)
		require.InDelta(t, 0.5, got.Rate, 0.001)
	})

	t.Run("bool field gets default", func(t *testing.T) {
		type cfg struct {
			Debug bool `toml:"debug" config:"default=true"`
		}
		got, err := LoadBytes[cfg]([]byte(""), TOML)
		require.NoError(t, err)
		require.True(t, got.Debug)
	})

	t.Run("duration field gets default", func(t *testing.T) {
		type cfg struct {
			Timeout time.Duration `toml:"timeout" config:"default=5s"`
		}
		got, err := LoadBytes[cfg]([]byte(""), TOML)
		require.NoError(t, err)
		require.Equal(t, 5*time.Second, got.Timeout)
	})

	t.Run("explicit value is not overwritten by default", func(t *testing.T) {
		type cfg struct {
			Port int `toml:"port" config:"default=8080"`
		}
		got, err := LoadBytes[cfg]([]byte("port = 9090"), TOML)
		require.NoError(t, err)
		require.Equal(t, 9090, got.Port)
	})
}

func TestLoadBytes_ValidatesRequired(t *testing.T) {
	t.Run("empty string fails required", func(t *testing.T) {
		type cfg struct {
			Name string `toml:"name" config:"required"`
		}
		_, err := LoadBytes[cfg]([]byte(""), TOML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Name")
	})

	t.Run("set string passes required", func(t *testing.T) {
		type cfg struct {
			Name string `toml:"name" config:"required"`
		}
		_, err := LoadBytes[cfg]([]byte("name = \"hello\""), TOML)
		require.NoError(t, err)
	})

	t.Run("nil slice fails required", func(t *testing.T) {
		type cfg struct {
			Items []string `toml:"items" config:"required"`
		}
		_, err := LoadBytes[cfg]([]byte(""), TOML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Items")
	})
}

func TestLoadBytes_ValidatesNumericBounds(t *testing.T) {
	type minCfg struct {
		Count int `toml:"count" config:"min=1"`
	}
	type maxCfg struct {
		Count int `toml:"count" config:"max=100"`
	}

	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{
			name: "value below min is rejected",
			run: func() error {
				_, err := LoadBytes[minCfg]([]byte("count = 0"), TOML)
				return err
			},
			wantErr: true,
		},
		{
			name: "value at min is accepted",
			run: func() error {
				_, err := LoadBytes[minCfg]([]byte("count = 1"), TOML)
				return err
			},
		},
		{
			name: "value above min is accepted",
			run: func() error {
				_, err := LoadBytes[minCfg]([]byte("count = 5"), TOML)
				return err
			},
		},
		{
			name: "value above max is rejected",
			run: func() error {
				_, err := LoadBytes[maxCfg]([]byte("count = 101"), TOML)
				return err
			},
			wantErr: true,
		},
		{
			name: "value at max is accepted",
			run: func() error {
				_, err := LoadBytes[maxCfg]([]byte("count = 100"), TOML)
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
	t.Run("string shorter than min is rejected", func(t *testing.T) {
		type cfg struct {
			Code string `toml:"code" config:"min=3"`
		}
		_, err := LoadBytes[cfg]([]byte("code = \"ab\""), TOML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Code")
	})

	t.Run("string at min is accepted", func(t *testing.T) {
		type cfg struct {
			Code string `toml:"code" config:"min=3"`
		}
		_, err := LoadBytes[cfg]([]byte("code = \"abc\""), TOML)
		require.NoError(t, err)
	})

	t.Run("string longer than max is rejected", func(t *testing.T) {
		type cfg struct {
			Code string `toml:"code" config:"max=5"`
		}
		_, err := LoadBytes[cfg]([]byte("code = \"abcdef\""), TOML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Code")
	})

	t.Run("string at max is accepted", func(t *testing.T) {
		type cfg struct {
			Code string `toml:"code" config:"max=5"`
		}
		_, err := LoadBytes[cfg]([]byte("code = \"abcde\""), TOML)
		require.NoError(t, err)
	})
}

func TestLoadBytes_ValidatesNonempty(t *testing.T) {
	t.Run("empty string is rejected", func(t *testing.T) {
		type cfg struct {
			Name string `toml:"name" config:"nonempty"`
		}
		_, err := LoadBytes[cfg]([]byte("name = \"\""), TOML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Name")
	})

	t.Run("empty slice is rejected", func(t *testing.T) {
		type cfg struct {
			Items []string `toml:"items" config:"nonempty"`
		}
		_, err := LoadBytes[cfg]([]byte("items = []"), TOML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Items")
	})

	t.Run("non-empty slice is accepted", func(t *testing.T) {
		type cfg struct {
			Items []string `toml:"items" config:"nonempty"`
		}
		_, err := LoadBytes[cfg]([]byte("items = [\"a\"]"), TOML)
		require.NoError(t, err)
	})
}

func TestLoadBytes_ValidatesOneof(t *testing.T) {
	type cfg struct {
		Mode string `toml:"mode" config:"oneof=a|b|c"`
	}

	t.Run("value in set is accepted", func(t *testing.T) {
		_, err := LoadBytes[cfg]([]byte("mode = \"b\""), TOML)
		require.NoError(t, err)
	})

	t.Run("value not in set is rejected", func(t *testing.T) {
		_, err := LoadBytes[cfg]([]byte("mode = \"d\""), TOML)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Mode")
	})
}

func TestLoadBytes_CollectsAllValidationErrors(t *testing.T) {
	type cfg struct {
		A string `toml:"a" config:"required"`
		B string `toml:"b" config:"required"`
		C string `toml:"c" config:"required"`
	}

	_, err := LoadBytes[cfg]([]byte(""), TOML)
	require.Error(t, err)

	msg := err.Error()
	require.Contains(t, msg, "A")
	require.Contains(t, msg, "B")
	require.Contains(t, msg, "C")
}

func TestLoadBytes_ValidatesNestedStructFields(t *testing.T) {
	type dbCfg struct {
		Primary string `toml:"primary" config:"required"`
	}
	type cfg struct {
		Database dbCfg `toml:"database"`
	}

	_, err := LoadBytes[cfg]([]byte("[database]\nprimary = \"\""), TOML)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Database.Primary")
}

func TestLoadBytes_CallsValidatorInterface(t *testing.T) {
	_, err := LoadBytes[validatedCfg]([]byte("port = 0"), TOML)
	require.Error(t, err)
	require.Contains(t, err.Error(), "port must be positive")
}

// validatedCfg is defined at file scope so the Validate method can be declared.
type validatedCfg struct {
	Port int `toml:"port"`
}

func (c *validatedCfg) Validate() error {
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	return nil
}

func TestLoadBytes_ExpandsEnvVars(t *testing.T) {
	type cfg struct {
		Secret string `toml:"secret"`
	}

	t.Setenv("CONFIG_TEST_SECRET", "hunter2")

	got, err := LoadBytes[cfg]([]byte("secret = \"${CONFIG_TEST_SECRET}\""), TOML)
	require.NoError(t, err)
	require.Equal(t, "hunter2", got.Secret)
}

func TestLoadBytes_ExpandsEnvVarsWithDefault(t *testing.T) {
	type cfg struct {
		Host string `toml:"host"`
		Port string `toml:"port"`
	}

	t.Run("uses default when env var is unset", func(t *testing.T) {
		got, err := LoadBytes[cfg]([]byte("host = \"${CONFIG_TEST_UNSET_VAR:-localhost}\""), TOML)
		require.NoError(t, err)
		require.Equal(t, "localhost", got.Host)
	})

	t.Run("uses default when env var is empty", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_EMPTY_VAR", "")

		got, err := LoadBytes[cfg]([]byte("host = \"${CONFIG_TEST_EMPTY_VAR:-fallback}\""), TOML)
		require.NoError(t, err)
		require.Equal(t, "fallback", got.Host)
	})

	t.Run("uses env value when set", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_SET_VAR", "prod.example.com")

		got, err := LoadBytes[cfg]([]byte("host = \"${CONFIG_TEST_SET_VAR:-localhost}\""), TOML)
		require.NoError(t, err)
		require.Equal(t, "prod.example.com", got.Host)
	})

	t.Run("empty default is valid", func(t *testing.T) {
		got, err := LoadBytes[cfg]([]byte("host = \"${CONFIG_TEST_UNSET_VAR2:-}\""), TOML)
		require.NoError(t, err)
		require.Equal(t, "", got.Host)
	})
}

func TestLoad_DetectsFormatFromExtension(t *testing.T) {
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
		type cfg struct {
			Host string `toml:"host"`
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "config.txt")
		require.NoError(t, os.WriteFile(path, []byte("host = \"example.com\""), 0o644))

		_, err := Load[cfg](path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported")
	})
}
