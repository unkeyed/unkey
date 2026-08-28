package deploymentgc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRegistryProjectID(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		want       string
		wantErr    bool
	}{
		{name: "valid", repository: "registry.depot.dev/abc123", want: "abc123"},
		{name: "empty", repository: "", wantErr: true},
		{name: "wrong_host", repository: "example.com/abc123", wantErr: true},
		{name: "missing_project", repository: "registry.depot.dev/", wantErr: true},
		{name: "nested_path", repository: "registry.depot.dev/team/abc123", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseRegistryProjectID(test.repository)
			if test.wantErr {
				require.Error(t, err, "ParseRegistryProjectID(%q)", test.repository)
				return
			}
			require.NoError(t, err, "ParseRegistryProjectID(%q)", test.repository)
			require.Equal(t, test.want, got, "ParseRegistryProjectID(%q)", test.repository)
		})
	}
}

func TestManagedImageDeploymentID(t *testing.T) {
	tests := []struct {
		name             string
		tag              string
		wantDeploymentID string
		wantOK           bool
	}{
		{name: "valid", tag: "proj_projectgc-d_reference", wantDeploymentID: "d_reference", wantOK: true},
		{name: "missing_separator", tag: "proj_projectgc"},
		{name: "extra_separator", tag: "proj_projectgc-d_reference-extra"},
		{name: "wrong_project_prefix", tag: "app_projectgc-d_reference"},
		{name: "wrong_deployment_prefix", tag: "proj_projectgc-dep_reference"},
		{name: "invalid_character", tag: "proj_projectgc-d_invalid!"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotDeploymentID, gotOK := managedImageDeploymentID(test.tag)
			require.Equal(t, test.wantOK, gotOK, "managedImageDeploymentID(%q) validity", test.tag)
			require.Equal(t, test.wantDeploymentID, gotDeploymentID, "managedImageDeploymentID(%q)", test.tag)
		})
	}
}
