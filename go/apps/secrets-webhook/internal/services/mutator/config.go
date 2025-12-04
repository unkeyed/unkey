package mutator

import "fmt"

const (
	unkeyEnvVolumeName      = "unkey-env-bin"
	unkeyEnvMountPath       = "/unkey"
	unkeyEnvBinary          = "/unkey/unkey-env"
	ServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

const (
	AnnotationDeploymentID     = "deployment-id"
	AnnotationProviderEndpoint = "provider-endpoint"
)

type Config struct {
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
	cfg := &podConfig{}

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
