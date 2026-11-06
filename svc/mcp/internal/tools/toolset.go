package tools

// ToolsetID names a toolset in the x-unkey-mcp spec extension and in
// analytics rows.
type ToolsetID string

const (
	ToolsetV2      ToolsetID = "v2"
	ToolsetCompute ToolsetID = "compute"
)

// Toolset is one MCP server that svc/mcp exposes. Path is the canonical URL
// path and LegacyPath the Speakeasy-era one. Both serve MCP traffic, and each
// is its own OAuth protected resource because clients bind tokens to the URL
// they connected to.
type Toolset struct {
	ID         ToolsetID
	Title      string
	Path       string
	LegacyPath string
}

// Toolsets is the only list of toolsets. [Load] rejects spec extensions that
// name any other ID, and the gateway derives every route from it.
var Toolsets = []Toolset{
	{ID: ToolsetV2, Title: "Unkey", Path: "/", LegacyPath: "/mcp/v2"},
	{ID: ToolsetCompute, Title: "Unkey Compute", Path: "/compute", LegacyPath: "/mcp/compute"},
}

func knownToolset(id ToolsetID) bool {
	for _, t := range Toolsets {
		if t.ID == id {
			return true
		}
	}
	return false
}
