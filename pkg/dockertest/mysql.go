package dockertest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

const (
	mysqlImage    = "mysql:9.4.0"
	mysqlPort     = "3306/tcp"
	mysqlUser     = "unkey"
	mysqlPassword = "password"
	mysqlDatabase = "unkey"
)

// MySQLConfig holds connection information for a MySQL test container.
type MySQLConfig struct {
	// DSN is the host DSN for connecting from the test runner.
	DSN string
	// DockerDSN is the DSN for connecting from containers on the docker network.
	DockerDSN string
}

// MySQL starts the local MySQL test container and returns DSNs.
//
// The container is based on the local dev image with preloaded schema.
// This function blocks until the MySQL port is accepting TCP connections
// (up to 60s). Fails the test if Docker is unavailable or the container fails to start.
func MySQL(t *testing.T) MySQLConfig {
	t.Helper()

	ctr := startContainer(t, containerConfig{
		Image:        mysqlImage,
		ExposedPorts: []string{mysqlPort},
		WaitStrategy: NewTCPWait(mysqlPort),
		WaitTimeout:  60 * time.Second,
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": mysqlPassword,
			"MYSQL_DATABASE":      mysqlDatabase,
			"MYSQL_USER":          mysqlUser,
			"MYSQL_PASSWORD":      mysqlPassword,
		},
		Cmd: []string{},
	})

	port := ctr.Port(mysqlPort)
	addr := fmt.Sprintf("%s:%s", ctr.Host, port)

	hostCfg := mysql.NewConfig()
	hostCfg.User = mysqlUser
	hostCfg.Passwd = mysqlPassword
	hostCfg.Net = "tcp"
	hostCfg.Addr = addr
	hostCfg.DBName = mysqlDatabase
	hostCfg.ParseTime = true
	hostCfg.MultiStatements = true
	hostCfg.Logger = &mysql.NopLogger{}

	dockerCfg := mysql.NewConfig()
	dockerCfg.User = mysqlUser
	dockerCfg.Passwd = mysqlPassword
	dockerCfg.Net = "tcp"
	dockerCfg.Addr = "mysql:3306"
	dockerCfg.DBName = mysqlDatabase
	dockerCfg.ParseTime = true
	dockerCfg.Logger = &mysql.NopLogger{}

	hostDB, err := sql.Open("mysql", hostCfg.FormatDSN())
	require.NoError(t, err)
	defer func() { require.NoError(t, hostDB.Close()) }()
	require.Eventually(t, func() bool {
		pingErr := hostDB.PingContext(context.Background())
		return pingErr == nil
	}, 60*time.Second, 500*time.Millisecond)

	schemaPath := schemaSQLPath()
	schemaBytes, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	_, err = hostDB.ExecContext(context.Background(), string(schemaBytes))
	require.NoError(t, err)

	return MySQLConfig{
		DSN:       hostCfg.FormatDSN(),
		DockerDSN: dockerCfg.FormatDSN(),
	}
}

func schemaSQLPath() string {
	if runfiles := os.Getenv("TEST_SRCDIR"); runfiles != "" {
		workspace := os.Getenv("TEST_WORKSPACE")
		if workspace != "" {
			candidate := filepath.Join(runfiles, workspace, "pkg", "db", "schema.sql")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		candidate := filepath.Join(runfiles, "_main", "pkg", "db", "schema.sql")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Dir(filepath.Dir(currentFile))
	return filepath.Join(root, "db", "schema.sql")
}
