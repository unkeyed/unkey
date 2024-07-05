package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type S3 struct {
	URL             string
	AccessKeyId     string
	AccessKeySecret string
	Stop            func()
}

// NewS3 runs a minion container and returns the URL
// The caller is responsible for stopping the container when done.
func NewS3(t *testing.T) S3 {

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
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

	url := fmt.Sprintf("http://%s:%s", host, port.Port())

	return S3{
		URL:             url,
		AccessKeyId:     "minio_root_user",
		AccessKeySecret: "minio_root_password",
		Stop: func() {
			require.NoError(t, container.Terminate(ctx))
		},
	}
}
