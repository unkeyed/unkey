package deploy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseForkRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		branch     string
		wantOwner  string
		wantBranch string
		wantOK     bool
	}{
		{
			name:       "fork ref with slash in branch",
			branch:     "alice-dev:feat/redesign-navbar",
			wantOwner:  "alice-dev",
			wantBranch: "feat/redesign-navbar",
			wantOK:     true,
		},
		{
			name:       "fork ref with simple branch",
			branch:     "contributor:feature-branch",
			wantOwner:  "contributor",
			wantBranch: "feature-branch",
			wantOK:     true,
		},
		{
			name:   "plain branch",
			branch: "main",
			wantOK: false,
		},
		{
			name:   "branch with slash",
			branch: "feature/something",
			wantOK: false,
		},
		{
			name:   "empty string",
			branch: "",
			wantOK: false,
		},
		{
			name:   "empty owner",
			branch: ":branch",
			wantOK: false,
		},
		{
			name:   "empty branch after colon",
			branch: "owner:",
			wantOK: false,
		},
		{
			name:   "slash before colon rejected",
			branch: "org/owner:branch",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			owner, branch, ok := parseForkRef(tt.branch)
			require.Equal(t, tt.wantOK, ok)
			if ok {
				require.Equal(t, tt.wantOwner, owner)
				require.Equal(t, tt.wantBranch, branch)
			}
		})
	}
}
