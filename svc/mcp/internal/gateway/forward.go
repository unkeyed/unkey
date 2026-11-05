package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/unkeyed/unkey/svc/mcp/internal/caller"
	"github.com/unkeyed/unkey/svc/mcp/internal/tools"
)

// responseBytesMax caps how much of an API response is returned to the model.
const responseBytesMax = 4 << 20

// outcome classifies a tool call for analytics.
type outcome string

const (
	outcomeOK             outcome = "ok"
	outcomeAPIError       outcome = "api_error"
	outcomeTransportError outcome = "transport_error"
	// outcomeRejected means svc/mcp refused the arguments and never called
	// the API.
	outcomeRejected outcome = "rejected"
)

// apiResponse is what one tool call produced. Status is 0 when no HTTP
// response arrived, and err then says why.
type apiResponse struct {
	outcome   outcome
	status    int
	body      []byte
	requestID string
	errorType string
	err       error
}

// callTool is the only tool handler: every tool is this function bound to a
// different [tools.Tool]. API failures become tool results the model can read
// and react to, never Go errors, which clients surface as protocol failures.
func (g *gateway) callTool(m mount, tool tools.Tool) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if req.Extra == nil {
			return nil, errors.New("tool call without HTTP request headers")
		}
		credential, err := caller.Parse(req.Extra.Header)
		if err != nil {
			return nil, fmt.Errorf("gate admitted a request without a usable credential: %w", err)
		}

		start := g.cfg.Clock.Now()
		response := g.call(ctx, credential, tool, req.Params.Arguments)
		g.cfg.Calls.Buffer(callRow(m, tool, credential, req, response, start, g.cfg.Clock.Now().Sub(start)))
		if digest, ok := credential.SessionDigest(); ok && response.status == http.StatusUnauthorized {
			g.cfg.Rejected.Set(ctx, digest, struct{}{})
		}
		return response.result(), nil
	}
}

func (g *gateway) call(ctx context.Context, credential caller.Credential, tool tools.Tool, arguments json.RawMessage) apiResponse {
	body, err := requestBody(tool, arguments)
	if err != nil {
		return apiResponse{outcome: outcomeRejected, status: 0, body: nil, requestID: "", errorType: "", err: err}
	}
	return g.forward(ctx, credential, tool, body)
}

// requestBody turns tool arguments into the API request body. Arguments are
// forwarded byte for byte: the API validates them against the same spec, and
// its 400 lists every violation for the model to fix.
func requestBody(tool tools.Tool, arguments json.RawMessage) ([]byte, error) {
	trimmed := bytes.TrimSpace(arguments)
	if !tool.WrapsBody && (len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))) {
		return []byte("{}"), nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return nil, errors.New("arguments must be a JSON object")
	}
	if !tool.WrapsBody {
		return trimmed, nil
	}
	body, ok := fields[tools.WrappedBodyProperty]
	if !ok {
		return nil, fmt.Errorf("arguments must have a %q property holding the request body", tools.WrappedBodyProperty)
	}
	return body, nil
}

func (g *gateway) forward(ctx context.Context, credential caller.Credential, tool tools.Tool, body []byte) apiResponse {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(g.cfg.APIURL, "/")+tool.Path, bytes.NewReader(body))
	if err != nil {
		return failedResponse(0, err)
	}
	req.Header.Set("Authorization", credential.Authorization())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "unkey-mcp/"+g.cfg.Version)

	res, err := g.cfg.HTTPClient.Do(req)
	if err != nil {
		return failedResponse(0, err)
	}
	defer func() { _ = res.Body.Close() }()

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, responseBytesMax+1))
	if err != nil {
		return failedResponse(res.StatusCode, err)
	}
	if len(responseBody) > responseBytesMax {
		return failedResponse(res.StatusCode, fmt.Errorf("the API response exceeded %d bytes", responseBytesMax))
	}

	var envelope struct {
		Meta struct {
			RequestID string `json:"requestId"`
		} `json:"meta"`
		Error *struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	response := apiResponse{outcome: outcomeOK, status: res.StatusCode, body: responseBody, requestID: "", errorType: "", err: nil}
	if res.StatusCode >= http.StatusBadRequest {
		response.outcome = outcomeAPIError
	}
	if json.Unmarshal(responseBody, &envelope) == nil {
		response.requestID = envelope.Meta.RequestID
		if envelope.Error != nil {
			response.errorType = envelope.Error.Type
		}
	}
	return response
}

func failedResponse(status int, err error) apiResponse {
	return apiResponse{outcome: outcomeTransportError, status: status, body: nil, requestID: "", errorType: "", err: err}
}

// result returns the API's JSON envelope verbatim, including meta.requestId
// and the error object, so the model sees what an SDK user would.
func (r apiResponse) result() *mcp.CallToolResult {
	switch r.outcome {
	case outcomeRejected:
		return errorResult("Invalid tool arguments: " + r.err.Error())
	case outcomeTransportError:
		return errorResult("The Unkey API call failed: " + r.err.Error())
	case outcomeOK, outcomeAPIError:
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(r.body)}},
			IsError: r.outcome == outcomeAPIError,
		}
	default:
		return errorResult(fmt.Sprintf("unknown tool call outcome %q", r.outcome))
	}
}

func errorResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}, IsError: true}
}
