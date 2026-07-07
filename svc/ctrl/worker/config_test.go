package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
)

func TestConfigDatabase(t *testing.T) {
	t.Run("accepts flat database string", func(t *testing.T) {
		cfg, err := config.LoadBytes[Config]([]byte(`
cname_domain = "unkey.local"
database = "unkey:password@tcp(mysql:3306)/unkey?parseTime=true"

[vault]
url = "http://vault:8060"
token = "vault-token"
`))

		require.NoError(t, err)
		require.Equal(t, "unkey:password@tcp(mysql:3306)/unkey?parseTime=true", cfg.Database)
	})

	t.Run("rejects nested database table", func(t *testing.T) {
		_, err := config.LoadBytes[Config]([]byte(`
cname_domain = "unkey.local"

[database]
primary = "unkey:password@tcp(mysql:3306)/unkey?parseTime=true"

[vault]
url = "http://vault:8060"
token = "vault-token"
`))

		require.Error(t, err)
	})
}
