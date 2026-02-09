package mutator

import (
	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/registry"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/registry/credentials"
)

const (
	injectVolumeName = "inject-bin"
	injectMountPath  = "/inject-bin"
	injectBinary     = "/inject-bin/inject"
	//nolint:gosec // G101: This is a file path, not credentials
	ServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

const (
	LabelInject       = "unkey.com/inject"
	LabelDeploymentID = "unkey.com/deployment.id"
	LabelBuildID      = "unkey.com/build.id"
)

type Config struct {
	Registry                *registry.Registry
	Clientset               kubernetes.Interface
	Credentials             *credentials.Manager
	InjectImage             string
	InjectImagePullPolicy   string
	DefaultProviderEndpoint string
}

type podConfig struct {
	DeploymentID     string
	ProviderEndpoint string
}

func (m *Mutator) loadPodConfig(labels map[string]string) (*podConfig, error) {
	cfg := &podConfig{
		DeploymentID:     labels[LabelDeploymentID],
		ProviderEndpoint: m.defaultProviderEndpoint,
	}

	if err := assert.All(
		assert.NotEmpty(cfg.DeploymentID, "missing required label: "+LabelDeploymentID),
	); err != nil {
		return nil, err
	}

	return cfg, nil
}
