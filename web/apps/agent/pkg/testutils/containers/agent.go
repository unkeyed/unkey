package containers

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Agent struct {
	URL string
}

// NewAgent runs an Agent container
// The caller is responsible for stopping the container when done.
func NewAgent(t *testing.T, clusterSize int) []Agent {
	t.Helper()

	ctx := context.Background()

	net, err := network.New(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, net.Remove(ctx))
	})

	s3 := NewS3(t, net.Name)
	t.Cleanup(s3.Stop)

	require.NoError(t, err)
	dockerContext := path.Join(os.Getenv("OLDPWD"), "./apps/agent")
	t.Logf("using docker context: %s", dockerContext)

	t.Log("s3 url: " + s3.InternalURL)
	agents := []Agent{}
	for i := 1; i <= clusterSize; i++ {

		agent, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Name:       fmt.Sprintf("unkey-agent-%d", i),
				SkipReaper: true,
				Networks:   []string{net.Name},
				FromDockerfile: testcontainers.FromDockerfile{
					Context:    dockerContext,
					Dockerfile: "Dockerfile",
				},
				Cmd:          []string{"/usr/local/bin/unkey", "agent", "--config", "config.docker.json"},
				ExposedPorts: []string{"8081/tcp"},

				Env: map[string]string{
					"PORT":                       "8081",
					"SERF_PORT":                  "9090",
					"RPC_PORT":                   "9095",
					"AUTH_TOKEN":                 "agent-auth-secret",
					"VAULT_S3_URL":               s3.InternalURL,
					"VAULT_S3_BUCKET":            "vault",
					"VAULT_S3_ACCESS_KEY_ID":     s3.AccessKeyId,
					"VAULT_S3_ACCESS_KEY_SECRET": s3.AccessKeySecret,
					"VAULT_MASTER_KEYS":          "Ch9rZWtfMmdqMFBJdVhac1NSa0ZhNE5mOWlLSnBHenFPENTt7an5MRogENt9Si6wms4pQ2XIvqNSIgNpaBenJmXgcInhu6Nfv2U=",
				},
				WaitingFor: wait.ForHTTP("/v1/liveness"),
			},
		})

		require.NoError(t, err)

		err = agent.Start(ctx)
		require.NoError(t, err)
		// t.Cleanup(func() {
		// 	require.NoError(t, agent.Terminate(ctx))
		// })
		t.Log(agent.Networks(ctx))

		host, err := agent.Host(ctx)
		require.NoError(t, err)

		port, err := agent.MappedPort(ctx, "8081")
		require.NoError(t, err)

		url := fmt.Sprintf("http://%s:%s", host, port.Port())

		require.NotEmpty(t, url, "connection string is empty")

		agents = append(agents, Agent{
			URL: url,
		})
	}

	return agents

}
