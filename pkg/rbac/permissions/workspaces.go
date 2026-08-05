package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// InstallGithub authorizes installing the Unkey GitHub App for a workspace.
type InstallGithub struct{}

func (InstallGithub) ActionFor(urn.Workspace) {}
func (InstallGithub) String() string          { return "install_github" }
