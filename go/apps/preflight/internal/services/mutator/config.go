package mutator

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/go/apps/preflight/internal/services/registry"
	"github.com/unkeyed/unkey/go/apps/preflight/internal/services/registry/credentials"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const (
	unkeyEnvVolumeName = "unkey-env-bin"
	unkeyEnvMountPath  = "/unkey"
	unkeyEnvBinary     = "/unkey/unkey-env"
	//nolint:gosec // G101: This is a file path, not credentials
	ServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

const (
	AnnotationDeploymentID     = "deployment-id"
	AnnotationProviderEndpoint = "provider-endpoint"
)

type Config struct {
	Logger                  logging.Logger
	Registry                *registry.Registry
	Clientset               kubernetes.Interface
	Credentials             *credentials.Manager
	UnkeyEnvImage           string
	UnkeyEnvImagePullPolicy string
	AnnotationPrefix        string
	DefaultProviderEndpoint string
}

func (c *Config) GetAnnotation(suffix string) string {
	return fmt.Sprintf("%s/%s", c.AnnotationPrefix, suffix)
}

type podConfig struct {
	DeploymentID     string
	ProviderEndpoint string
}

func (m *Mutator) loadPodConfig(annotations map[string]string) (*podConfig, error) {
	cfg := &podConfig{
		DeploymentID:     "",
		ProviderEndpoint: "",
	}

	cfg.DeploymentID = annotations[m.cfg.GetAnnotation(AnnotationDeploymentID)]
	if cfg.DeploymentID == "" {
		return nil, fmt.Errorf("missing required annotation: %s", m.cfg.GetAnnotation(AnnotationDeploymentID))
	}

	if val, ok := annotations[m.cfg.GetAnnotation(AnnotationProviderEndpoint)]; ok && val != "" {
		cfg.ProviderEndpoint = val
	} else {
		cfg.ProviderEndpoint = m.cfg.DefaultProviderEndpoint
	}

	return cfg, nil
}
