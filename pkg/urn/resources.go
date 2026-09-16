package urn

// New returns builders for canonical Unkey resource names.
func New() Builder {
	return Builder{}
}

// Builder starts typed URN construction.
type Builder struct{}

// Workspace returns builders for canonical v1 resource names in a workspace.
//
// Workspace is the root of every v1 Unkey resource name:
//
//	unkey:v1:{workspace_id}:{resource_path}
//
// Every builder below this point stays inside this workspace. A wildcard such
// as "**" can cover every resource in this workspace, but never crosses into
// another workspace.
func (Builder) Workspace(workspaceID string) workspace {
	return workspace{workspaceID: workspaceID}
}
