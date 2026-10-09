package privatenetwork

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/unkeyed/unkey/pkg/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	ConnectionSchemaVersion = "1"
	ConnectionAliasKey      = "appSlug"
	ConnectionDeploymentKey = "deploymentId"
	ConnectionServiceKey    = "serviceName"
	ConnectionRevisionKey   = "revision"
)

const schemaVersionKey = "schemaVersion"

// ConnectionData is the content of a published connection ConfigMap. An empty
// DeploymentID and ServiceName mean the connection has no target.
type ConnectionData struct {
	Alias        string
	DeploymentID string
	ServiceName  string
	Revision     uint64
}

// Encode validates data and returns it as ConfigMap data.
func Encode(data ConnectionData) (map[string]string, error) {
	if err := validateConnectionData(data); err != nil {
		return nil, err
	}
	return map[string]string{
		ConnectionAliasKey: data.Alias, ConnectionDeploymentKey: data.DeploymentID, ConnectionServiceKey: data.ServiceName,
		ConnectionRevisionKey: strconv.FormatUint(data.Revision, 10), schemaVersionKey: ConnectionSchemaVersion,
	}, nil
}

// Decode parses and validates ConfigMap data written by [Encode].
func Decode(values map[string]string) (ConnectionData, error) {
	if version := values[schemaVersionKey]; version != ConnectionSchemaVersion {
		return ConnectionData{}, fmt.Errorf("unsupported connection schema version %q", version)
	}
	revision, err := strconv.ParseUint(values[ConnectionRevisionKey], 10, 64)
	if err != nil {
		return ConnectionData{}, fmt.Errorf("invalid connection revision: %w", err)
	}
	data := ConnectionData{Alias: values[ConnectionAliasKey], DeploymentID: values[ConnectionDeploymentKey], ServiceName: values[ConnectionServiceKey], Revision: revision}
	if err := validateConnectionData(data); err != nil {
		return ConnectionData{}, err
	}
	return data, nil
}

func validateConnectionData(data ConnectionData) error {
	return assert.All(
		assert.True(len(validation.IsDNS1123Label(data.Alias)) == 0, fmt.Sprintf("invalid connection alias %q", data.Alias)),
		assert.Equal(data.DeploymentID == "", data.ServiceName == "", "connection target is only partially resolved"),
		assert.True(data.ServiceName == "" || len(validation.IsDNS1123Label(data.ServiceName)) == 0, fmt.Sprintf("invalid connection service name %q", data.ServiceName)),
		assert.NotZero(data.Revision, "connection revision must be positive"),
	)
}

// ServiceName returns the discovery Service name shared by connections to one
// deployment port.
func ServiceName(deploymentID string, port int32) string {
	sum := sha256.Sum256([]byte(deploymentID + "/" + strconv.Itoa(int(port))))
	return "unkey-private-" + hex.EncodeToString(sum[:16])
}
