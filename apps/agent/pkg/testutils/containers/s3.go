package containers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type S3 struct {
	URL string
	// From another container
	InternalURL     string
	AccessKeyId     string
	AccessKeySecret string
	Stop            func()
}

// NewS3 runs a minion container and returns the URL
// The caller is responsible for stopping the container when done.
func NewS3(t *testing.T, networks ...string) S3 {

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Name:         "s3",
		SkipReaper:   true,
		Networks:     networks,
		Image:        "bitnamilegacy/minio:2025.7.23-debian-12-r5",
		ExposedPorts: []string{"9000/tcp"},
		WaitingFor:   wait.ForHTTP("/minio/health/live").WithPort("9000"),
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minio_root_user",
			"MINIO_ROOT_PASSWORD": "minio_root_password",
		},
		Cmd: []string{"server", "/data"},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{

		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "9000")
	require.NoError(t, err)
	ip, err := container.ContainerIP(ctx)
	require.NoError(t, err)

	t.Log(container.Networks(ctx))
	name, err := container.Name(ctx)
	require.NoError(t, err)
	url := fmt.Sprintf("http://%s:%s", host, port.Port())
	t.Logf("S3 Name: %s", name)
	t.Logf("S3 IP: %s", ip)
	return S3{
		URL:             url,
		InternalURL:     fmt.Sprintf("http://%s:%s", strings.TrimPrefix(name, "/"), "9000"),
		AccessKeyId:     "minio_root_user",
		AccessKeySecret: "minio_root_password",
		Stop: func() {
			require.NoError(t, container.Terminate(ctx))
		},
	}
}
