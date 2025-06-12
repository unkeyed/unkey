package containers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/retry"
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
	password := "minio_root_password" // nolint:gosec

	runOpts := &dockertest.RunOptions{
		Name:       containerNameS3,
		Repository: "minio/minio",
		Tag:        "RELEASE.2025-04-03T14-56-28Z", // They fucked their license or something and it broke, don't use latest
		Networks:   []*dockertest.Network{c.network},
		Env: []string{
			fmt.Sprintf("MINIO_ROOT_USER=%s", user),
			fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", password),
		},
		Cmd: []string{"server", "/Data"},
	}

	resource, _, err := c.getOrCreateContainer(containerNameS3, runOpts)
	require.NoError(c.t, err)

	s3 := S3{
		DockerURL:       fmt.Sprintf("http://%s:9000", resource.GetIPInNetwork(c.network)),
		HostURL:         fmt.Sprintf("http://localhost:%s", resource.GetPort("9000/tcp")),
		AccessKeyId:     user,
		AccessKeySecret: password,
	}

	err = retry.New(
		retry.Attempts(10),
		retry.Backoff(func(n int) time.Duration {
			return time.Duration(n*n*100) * time.Millisecond
		}),
	).Do(func() error {
		resp, liveErr := http.Get(fmt.Sprintf("%s/minio/health/live", s3.HostURL))
		if liveErr != nil {
			return liveErr
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %s", resp.Status)
		}
		return nil
	})
	require.NoError(t, err, "S3 is not healthy")

	return s3
}
