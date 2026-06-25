package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatabaseConfigValidate(t *testing.T) {
	t.Run("accepts primary", func(t *testing.T) {
		cfg := DatabaseConfig{
			Primary: "unkey:password@tcp(mysql:3306)/unkey?parseTime=true",
		}

		require.NoError(t, cfg.Validate())
	})

	t.Run("rejects readonly replica", func(t *testing.T) {
		cfg := DatabaseConfig{
			Primary:         "unkey:password@tcp(mysql:3306)/unkey?parseTime=true",
			ReadonlyReplica: "unkey:password@tcp(mysql-ro:3306)/unkey?parseTime=true",
		}

		require.ErrorContains(t, cfg.Validate(), "database.readonly_replica is not supported")
	})
}
