package registrysweep

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentIDFromTag(t *testing.T) {
	tests := []struct {
		tag          string
		deploymentID string
		ok           bool
	}{
		{tag: "proj_abc123-d_xyz789", deploymentID: "d_xyz789", ok: true},
		{tag: "latest", ok: false},
		{tag: "v1.2.3", ok: false},
		{tag: "proj_abc123", ok: false},
		{tag: "proj_abc123-", ok: false},
		{tag: "proj_-d_xyz", ok: false},
		{tag: "proj_abc-d_", ok: false},
		{tag: "other_abc-d_xyz", ok: false},
		{tag: "proj_abc-dep_xyz", ok: false},
		{tag: "proj_abc-d_xyz-copy", ok: false},
		{tag: "proj_abc-d_xyz.other", ok: false},
		{tag: "proj_abc!-d_xyz", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			id, ok := deploymentIDFromTag(tt.tag)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.deploymentID, id)
		})
	}
}

func TestProjectIDFromDepotName(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		projectID string
		ok        bool
	}{
		{name: "builds-prod-proj_abc123", prefix: "builds-prod", projectID: "proj_abc123", ok: true},
		{name: "builds-preview-proj_abc123", prefix: "builds-prod", ok: false},
		{name: "builds-prod-somethingelse", prefix: "builds-prod", ok: false},
		{name: "builds-prod-proj_", prefix: "builds-prod", ok: false},
		{name: "builds-prod-proj_abc-copy", prefix: "builds-prod", ok: false},
		{name: "builds-prod-proj_abc.other", prefix: "builds-prod", ok: false},
		{name: "manually-created", prefix: "builds-prod", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := projectIDFromDepotName(tt.name, tt.prefix)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.projectID, id)
		})
	}
}

func TestDepotRepositoryPath(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		path       string
		wantErr    bool
	}{
		{name: "Depot repository", repository: "registry.depot.dev/project", path: "project"},
		{name: "nested Depot repository", repository: "registry.depot.dev/org/project", path: "org/project"},
		{name: "missing host", repository: "project", wantErr: true},
		{name: "missing path", repository: "registry.depot.dev/", wantErr: true},
		{name: "foreign registry", repository: "ghcr.io/org/project", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := depotRepositoryPath(tt.repository)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.path, path)
		})
	}
}
