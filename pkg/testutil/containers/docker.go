package containers

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/cpuguy83/dockercfg"
	"github.com/moby/moby/client"
	"go.opentelemetry.io/otel/attribute"
)

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

	if c, ok := reuseService(t, docker, service); ok {
		return c
	}
	return startComposeService(t, service)
}

func newDockerClient() (client.APIClient, error) {
	contextName := os.Getenv("DOCKER_CONTEXT")
	if contextName == "" && os.Getenv("DOCKER_HOST") == "" {
		configuration, err := dockercfg.LoadDefaultConfig()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		contextName = configuration.CurrentContext
	}
	if contextName != "" && contextName != "default" {
		return nil, errors.New("named Docker contexts require Compose")
	}
	if os.Getenv("DOCKER_TLS") != "" || os.Getenv("DOCKER_TLS_VERIFY") != "" || os.Getenv("DOCKER_CERT_PATH") != "" {
		return nil, errors.New("Docker TLS configuration requires Compose")
	}
	return client.New(client.FromEnv)
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
