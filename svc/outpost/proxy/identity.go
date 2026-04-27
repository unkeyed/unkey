package proxy

import (
	"encoding/base64"
	"net/http"
	"strings"
)

type EgressIdentity struct {
	WorkspaceID  string
	DeploymentID string
}

func ParseIdentity(r *http.Request) EgressIdentity {
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Basic ") {
		return EgressIdentity{WorkspaceID: "", DeploymentID: ""}
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len("Basic "):])
	if err != nil {
		return EgressIdentity{WorkspaceID: "", DeploymentID: ""}
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return EgressIdentity{WorkspaceID: "", DeploymentID: ""}
	}

	return EgressIdentity{
		WorkspaceID:  parts[0],
		DeploymentID: parts[1],
	}
}
