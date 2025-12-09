package credentials

import (
	"context"
	"encoding/base64"
	"encoding/json"
)

// Registry provides credentials for pulling images from a container registry.
type Registry interface {
	// Matches returns true if this registry handles the given image.
	Matches(image string) bool

	// GetCredentials returns Docker config credentials for the image.
	// buildID is optional and used for build-scoped tokens when available.
	GetCredentials(ctx context.Context, image, buildID string) (*DockerConfigJSON, error)
}

// DockerConfigJSON represents the Docker config.json format for registry auth.
type DockerConfigJSON struct {
	Auths map[string]DockerAuth `json:"auths"`
}

// DockerAuth represents authentication for a single registry.
type DockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

// ToJSON serializes the Docker config to JSON bytes suitable for a K8s secret.
func (d *DockerConfigJSON) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// Merge adds all auths from another DockerConfigJSON into this one.
func (d *DockerConfigJSON) Merge(other *DockerConfigJSON) {
	if other == nil {
		return
	}
	for registry, auth := range other.Auths {
		d.Auths[registry] = auth
	}
}

// NewDockerConfig creates a DockerConfigJSON for the given registry and credentials.
func NewDockerConfig(registry, username, password string) *DockerConfigJSON {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return &DockerConfigJSON{
		Auths: map[string]DockerAuth{
			registry: {
				Username: username,
				Password: password,
				Auth:     auth,
			},
		},
	}
}

// Manager holds multiple registries and finds credentials for images.
type Manager struct {
	registries []Registry
}

// NewManager creates a new registry manager.
func NewManager(registries ...Registry) *Manager {
	return &Manager{registries: registries}
}

// GetCredentials finds a matching registry and returns credentials for the image.
// Returns nil if no registry matches.
func (m *Manager) GetCredentials(ctx context.Context, image, buildID string) (*DockerConfigJSON, error) {
	for _, reg := range m.registries {
		if reg.Matches(image) {
			return reg.GetCredentials(ctx, image, buildID)
		}
	}

	return nil, nil
}

// Matches returns true if any registry can handle the image.
func (m *Manager) Matches(image string) bool {
	for _, reg := range m.registries {
		if reg.Matches(image) {
			return true
		}
	}

	return false
}

// NewDockerConfig creates an empty DockerConfigJSON for merging credentials.
func (m *Manager) NewDockerConfig() *DockerConfigJSON {
	return &DockerConfigJSON{
		Auths: make(map[string]DockerAuth),
	}
}
