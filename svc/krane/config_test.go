package krane

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestConfigDistinguishesUnsetAndEmptyRuntimeClass(t *testing.T) {
	for _, tt := range []struct {
		name    string
		setting string
		want    *string
	}{
		{name: "unset", setting: "", want: nil},
		{name: "node default", setting: "runtime_class_name = \"\"\n", want: ptr.P("")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.LoadBytes[Config]([]byte(tt.setting + `
[cluster]
cell_id = "local"
region = "local"
platform = "dev"
[control]
url = "http://ctrl"
token = "test"
[vault]
url = "http://vault"
token = "test"
`))
			require.NoError(t, err)
			require.Equal(t, tt.want, cfg.RuntimeClassName)
		})
	}
}
