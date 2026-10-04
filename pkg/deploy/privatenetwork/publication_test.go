package privatenetwork

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestConnectionData(t *testing.T) {
	want := ConnectionData{Alias: "payments-api", DeploymentID: uid.New(uid.DeploymentPrefix), ServiceName: "unkey-private-target", Revision: 7}

	encoded, err := Encode(want)
	require.NoError(t, err)
	require.Equal(t, "1", encoded["schemaVersion"])
	decoded, err := Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, want, decoded)
}

func TestUnresolvedConnectionDataRoundTrip(t *testing.T) {
	want := ConnectionData{Alias: "payments-api", Revision: 1}
	encoded, err := Encode(want)
	require.NoError(t, err)
	decoded, err := Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, want, decoded)
}

func TestDecodeConnectionDataRejectsMalformedValues(t *testing.T) {
	valid := map[string]string{"appSlug": "payments", "deploymentId": uid.New(uid.DeploymentPrefix), "serviceName": "target", "revision": "1", "schemaVersion": "1"}
	for name, mutate := range map[string]func(map[string]string){
		"missing version": func(data map[string]string) { delete(data, "schemaVersion") },
		"unknown version": func(data map[string]string) { data["schemaVersion"] = "2" },
		"invalid alias":   func(data map[string]string) { data["appSlug"] = "Payments" },
		"empty deployment": func(data map[string]string) {
			data["deploymentId"] = ""
		},
		"empty service only": func(data map[string]string) { data["serviceName"] = "" },
		"invalid service":    func(data map[string]string) { data["serviceName"] = "target.namespace" },
		"zero revision":      func(data map[string]string) { data["revision"] = "0" },
		"bad revision":       func(data map[string]string) { data["revision"] = "one" },
	} {
		t.Run(name, func(t *testing.T) {
			data := make(map[string]string, len(valid))
			for key, value := range valid {
				data[key] = value
			}
			mutate(data)
			_, err := Decode(data)
			require.Error(t, err)
		})
	}
}

func TestServiceName(t *testing.T) {
	require.Equal(t, "unkey-private-680e7f7bd57aa679f56c8c6d437f7091", ServiceName("dep_123", 8080))
}
