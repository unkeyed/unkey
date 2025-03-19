package containers

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

func (c *Containers) BuildAndRunAPI() {

	dsn := c.RunMySQL()

	// Get the path to the current file
	_, currentFilePath, _, _ := runtime.Caller(0)

	// Navigate from the current file to the project root (go/)
	// We're going from go/pkg/testutil/containers/ up to go/
	projectRoot := filepath.Join(filepath.Dir(currentFilePath), "../../../")

	// Define build options
	// nolint:exhaustruct
	buildOpts := &dockertest.BuildOptions{
		Dockerfile: "Dockerfile",
		ContextDir: projectRoot,
	}

	// Define run options
	// nolint:exhaustruct
	runOpts := &dockertest.RunOptions{
		// Configure your container run options
		Name: "api-test-container",
		Env: []string{
			//UNKEY_HTTP_PORT: 7070
			//   UNKEY_CLUSTER: true
			//   UNKEY_CLUSTER_GOSSIP_PORT: 9090
			//   UNKEY_CLUSTER_RPC_PORT: 9091
			//   # UNKEY_CLUSTER_ADVERTISE_ADDR_STATIC: "${HOSTNAME}"
			//   UNKEY_CLUSTER_DISCOVERY_REDIS_URL: "redis://redis:6379"
			//   UNKEY_DATABASE_PRIMARY_DSN: "mysql://unkey:password@tcp(mysql:3900)/unkey?parseTime=true
			fmt.Sprintf("UNKEY_DATABASE_PRIMARY_DSN=%s", dsn),
		},
		// Add other necessary options
	}

	_, err := c.pool.BuildAndRunWithBuildOptions(buildOpts, runOpts)
	require.NoError(c.t, err)

}
