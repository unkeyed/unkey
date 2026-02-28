package dockertest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	HostDSN   string
	DockerDSN string
}

// MySQL starts the local MySQL test container and returns DSNs.
func (c *Cluster) MySQL() MySQLConfig {
	c.t.Helper()

	ctr, cleanup, err := startContainer(c.cli, containerConfig{
		ContainerName: "",
		Image:         mysqlImage,
		ExposedPorts:  []string{mysqlPort},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": mysqlPassword,
			"MYSQL_DATABASE":      mysqlDatabase,
			"MYSQL_USER":          mysqlUser,
			"MYSQL_PASSWORD":      mysqlPassword,
		},
		Cmd: []string{
			"--skip-log-bin",
			"--disable-log-bin",
			"--innodb-doublewrite=0",
			"--innodb-flush-log-at-trx-commit=0",
			"--innodb-flush-method=nosync",
			"--innodb-buffer-pool-size=32M",
			"--innodb-log-buffer-size=1M",
			"--performance-schema=OFF",
			"--skip-name-resolve",
		},
		Tmpfs: map[string]string{
			"/var/lib/mysql": "rw,noexec,nosuid,size=256m",
		},
		Binds:       nil,
		Keep:        false,
		NetworkName: c.network.Name,
	}, c.t.Name())
	require.NoError(c.t, err)
	if cleanup != nil {
		c.t.Cleanup(func() { require.NoError(c.t, cleanup()) })
	}

	wait := NewTCPWait(mysqlPort)
	wait.Wait(c.t, ctr, 60*time.Second)

	hostCfg := mysqlHostConfig(ctr)

	hostDB, err := sql.Open("mysql", hostCfg.FormatDSN())
	require.NoError(c.t, err)
	defer func() { require.NoError(c.t, hostDB.Close()) }()
	require.Eventually(c.t, func() bool {
		pingErr := hostDB.PingContext(context.Background())
		return pingErr == nil
	}, 60*time.Second, 500*time.Millisecond)

	schemaPath := schemaSQLPath()
	schemaBytes, err := os.ReadFile(schemaPath)
	require.NoError(c.t, err)
	_, err = hostDB.ExecContext(context.Background(), string(schemaBytes))
	require.NoError(c.t, err)

	dockerCfg := mysql.NewConfig()
	dockerCfg.User = mysqlUser
	dockerCfg.Passwd = mysqlPassword
	dockerCfg.Net = "tcp"
	dockerCfg.Addr = fmt.Sprintf("%s:%s", ctr.ContainerName, containerPortNumber(mysqlPort))
	dockerCfg.DBName = mysqlDatabase
	dockerCfg.ParseTime = true
	dockerCfg.Logger = &mysql.NopLogger{}

	return MySQLConfig{
		HostDSN:   hostCfg.FormatDSN(),
		DockerDSN: dockerCfg.FormatDSN(),
	}
}

func mysqlHostConfig(ctr *Container) *mysql.Config {
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

	return hostCfg
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
