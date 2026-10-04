// Package cilium holds the CiliumNetworkPolicy pieces shared by Krane's
// deployment policies and private network connection policies, so both grant
// traffic to the same endpoints and ports.
package cilium

import (
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NetworkPolicyResource is the CiliumNetworkPolicy resource.
var NetworkPolicyResource = schema.GroupVersionResource{
	Group: "cilium.io", Version: "v2", Resource: "ciliumnetworkpolicies",
}

// DeploymentEndpoint selects the deployment Pods matching identity in every
// namespace and every ClusterMesh cluster.
func DeploymentEndpoint(identity labels.Labels) map[string]interface{} {
	matchLabels := make(map[string]interface{}, len(identity)+2)
	for key, value := range labels.New().ManagedByKrane().ComponentDeployment() {
		matchLabels[key] = value
	}
	for key, value := range identity {
		matchLabels[key] = value
	}
	return map[string]interface{}{
		"matchLabels": matchLabels,
		"matchExpressions": []interface{}{
			map[string]interface{}{"key": labels.LabelKeyNamespace, "operator": "Exists"},
			map[string]interface{}{"key": "io.cilium.k8s.policy.cluster", "operator": "Exists"},
		},
	}
}

// UnicastPorts allows every TCP and UDP port. An explicit range keeps ICMP,
// SCTP, and non-port protocols denied, which an omitted toPorts would allow.
func UnicastPorts() []interface{} {
	return []interface{}{map[string]interface{}{"ports": []interface{}{
		map[string]interface{}{"port": "1", "endPort": int64(65535), "protocol": "TCP"},
		map[string]interface{}{"port": "1", "endPort": int64(65535), "protocol": "UDP"},
	}}}
}
