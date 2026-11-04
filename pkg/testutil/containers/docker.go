package containers

import (
	"context"
	"crypto/sha256"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/moby/moby/client"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

var (
	reconciledServices sync.Map
	dockerClientMu     sync.Mutex
)

type serviceState struct {
	mu     sync.Mutex
	config [32]byte
}

func startService(t testing.TB, service string) Container {
	t.Helper()

	docker, err := newDockerClient()
	if err != nil {
		t.Logf("Docker SDK unavailable, using Compose: %v", err)
		return startComposeService(t, service)
	}
	defer func() {
		if err := docker.Close(); err != nil {
			t.Logf("close Docker SDK client: %v", err)
		}
	}()

	contents, err := os.ReadFile(composeFile())
	require.NoError(t, err)
	key := struct {
		project string
		service string
		host    string
	}{composeProjectName(), service, docker.DaemonHost()}
	var initial serviceState
	value, _ := reconciledServices.LoadOrStore(key, &initial)
	state := value.(*serviceState)
	state.mu.Lock()
	defer state.mu.Unlock()
	digest := sha256.Sum256(contents)

	// Reconcile each service through Compose before reusing it in this process.
	// The file digest prevents reuse after the Compose configuration changes.
	if state.config == digest {
		if c, ok := reuseService(t, docker, service); ok {
			return c
		}
	}
	c := startComposeService(t, service)
	state.config = digest
	return c
}

func newDockerClient() (client.APIClient, error) {
	// Docker CLI flag initialization writes package-global TLS defaults.
	dockerClientMu.Lock()
	defer dockerClientMu.Unlock()
	options := flags.NewClientOptions()
	flagSet := pflag.NewFlagSet("containers", pflag.ContinueOnError)
	options.InstallFlags(flagSet)
	options.SetDefaultOptions(flagSet)
	configuration, err := config.Load(config.Dir())
	if err != nil {
		return nil, err
	}
	return command.NewAPIClientFromFlags(options, configuration)
}

func reuseService(t testing.TB, docker client.APIClient, service string) (Container, bool) {
	t.Helper()
	span := startContainerSpan(t, "containers.docker.reuse", attribute.String("container.service", service))
	defer span.End()
	var unavailable Container
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	project := composeProjectName()
	var options client.ContainerListOptions
	options.Filters = client.Filters{}.
		Add("label", "com.docker.compose.project="+project, "com.docker.compose.service="+service, "com.docker.compose.oneoff=False").
		Add("status", "running").
		Add("health", "healthy")
	services, err := docker.ContainerList(ctx, options)
	if err != nil {
		t.Logf("Docker SDK lookup failed, using Compose: %v", err)
		return unavailable, false
	}
	if len(services.Items) != 1 {
		return unavailable, false
	}

	ports := make(map[int]int)
	for _, port := range services.Items[0].Ports {
		if port.Type == "tcp" && port.PublicPort != 0 {
			ports[int(port.PrivatePort)] = int(port.PublicPort)
		}
	}
	span.SetAttributes(attribute.Bool("container.reused", true))
	return Container{Name: service, project: project, ports: ports}, true
}
