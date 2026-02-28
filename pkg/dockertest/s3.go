package dockertest

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	minioImage = "quay.io/minio/minio:latest"
	minioPort  = 9000

	// Default MinIO credentials used for test containers.
	minioAccessKey = "minioadmin"
	minioSecretKey = "minioadmin"
)

// S3Config holds connection information for an S3-compatible test container.
type S3Config struct {
	HostURL         string
	DockerURL       string
	AccessKeyID     string
	SecretAccessKey string
}

// S3 starts a MinIO container and returns connection information.
func (c *Cluster) S3() S3Config {
	c.t.Helper()

	tcpPort := fmt.Sprintf("%d/tcp", minioPort)

	ctr, cleanup, err := startContainer(c.cli, containerConfig{
		ContainerName: "",
		Image:         minioImage,
		ExposedPorts:  []string{tcpPort},
		Env: map[string]string{
			"MINIO_ROOT_USER":     minioAccessKey,
			"MINIO_ROOT_PASSWORD": minioSecretKey,
		},
		Cmd:         []string{"server", "/data"},
		Tmpfs:       nil,
		Binds:       nil,
		Keep:        false,
		NetworkName: c.network.Name,
	}, c.t.Name())
	require.NoError(c.t, err)
	if cleanup != nil {
		c.t.Cleanup(func() { require.NoError(c.t, cleanup()) })
	}

	wait := NewHTTPWait(tcpPort, "/minio/health/live")
	wait.Wait(c.t, ctr, 30*time.Second)

	port := ctr.Port(tcpPort)
	require.NotEmpty(c.t, port, "s3 port not mapped")

	return S3Config{
		HostURL:         fmt.Sprintf("http://%s:%s", ctr.Host, port),
		DockerURL:       fmt.Sprintf("http://%s:%d", ctr.ContainerName, minioPort),
		AccessKeyID:     minioAccessKey,
		SecretAccessKey: minioSecretKey,
	}
}
