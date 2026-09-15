package containers

import (
	"testing"

	mysql "github.com/go-sql-driver/mysql"
)

// VitessConfig holds host connection addresses for the unsharded unkey keyspace.
type VitessConfig struct {
	Address string
	DSN     string
}

// Vitess starts the shared Vitess service with the repository schema and no seed data.
func Vitess(t testing.TB) VitessConfig {
	t.Helper()

	c := startService(t, "vitess")
	dsn := mysql.NewConfig()
	dsn.User = "unkey"
	dsn.Passwd = "password"
	dsn.Net = "tcp"
	dsn.Addr = c.Addr(t, 33577)
	dsn.DBName = "unkey"
	dsn.ParseTime = true
	dsn.MultiStatements = true

	return VitessConfig{
		Address: c.Addr(t, 33575),
		DSN:     dsn.FormatDSN(),
	}
}
