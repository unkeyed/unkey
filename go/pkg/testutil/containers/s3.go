package containers

import (
	"fmt"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

type S3 struct {
	// From another container
	DockerURL       string
	HostURL         string
	AccessKeyId     string
	AccessKeySecret string
}

// NewS3 runs a minion container and returns the URL
// The caller is responsible for stopping the container when done.
func (c *Containers) RunS3(t *testing.T) S3 {

	user := "minio_root_user"
	password := "minio_root_password"

	resource, err := c.pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "minio/minio",
		Tag:        "latest",
		Networks:   []*dockertest.Network{c.network},
		Env: []string{
			fmt.Sprintf("MINIO_ROOT_USER=%s", user),
			fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", password),
		},
		Cmd: []string{"server", "/Data"},
	})
	require.NoError(c.t, err)

	c.t.Cleanup(func() {
		require.NoError(c.t, c.pool.Purge(resource))
	})

	return S3{
		DockerURL:       fmt.Sprintf("http://%s:9000", resource.GetIPInNetwork(c.network)),
		HostURL:         fmt.Sprintf("http://localhost:%s", resource.GetPort("9000/tcp")),
		AccessKeyId:     user,
		AccessKeySecret: password,
	}
}
