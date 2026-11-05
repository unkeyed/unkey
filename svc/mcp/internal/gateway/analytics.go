package gateway

import (
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/mcp/internal/caller"
	"github.com/unkeyed/unkey/svc/mcp/internal/tools"
)

const clientFieldBytesMax = 64

// callRow builds the analytics row for one tools/call. It records
// identifiers and status only, never arguments or response bodies.
func callRow(m mount, tool tools.Tool, credential caller.Credential, req *mcp.CallToolRequest, response apiResponse, start time.Time, latency time.Duration) schema.McpToolCallV1 {
	clientName, clientVersion := clientIdentity(req)
	return schema.McpToolCallV1{
		RequestID:     uid.New(uid.RequestPrefix),
		Time:          start.UnixMilli(),
		Toolset:       string(m.toolset.ID),
		Mount:         m.path,
		Tool:          tool.Name,
		Outcome:       string(response.outcome),
		HttpStatus:    int32(response.status),
		ErrorType:     response.errorType,
		LatencyMs:     latency.Milliseconds(),
		AuthKind:      string(credential.Kind()),
		WorkspaceID:   credential.WorkspaceID(response.status),
		SessionID:     credential.SessionID(),
		ApiRequestID:  response.requestID,
		ClientName:    clientName,
		ClientVersion: clientVersion,
	}
}

// clientIdentity names the MCP client. Clients on protocol 2026-07-28 and
// later send clientInfo with every request. Older clients send it only at
// initialize, which a stateless server does not keep, so fall back to the
// first product token of the User-Agent, such as "claude-code/2.1.4".
func clientIdentity(req *mcp.CallToolRequest) (string, string) {
	if info := req.ClientInfo(); info != nil && info.Name != "" {
		return truncate(info.Name), truncate(info.Version)
	}
	product, _, _ := strings.Cut(req.Extra.Header.Get("User-Agent"), " ")
	name, version, _ := strings.Cut(product, "/")
	return truncate(name), truncate(version)
}

func truncate(value string) string {
	if len(value) <= clientFieldBytesMax {
		return value
	}
	return strings.ToValidUTF8(value[:clientFieldBytesMax], "")
}
