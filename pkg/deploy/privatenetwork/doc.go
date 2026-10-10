// Package privatenetwork defines the Kubernetes publication contract shared by
// Ctrl, Krane, and undns. It covers app connections and a deployment's own
// replicas, which are discoverable without a user-created connection.
//
// Ctrl selects deployment targets. Krane publishes ConfigMaps, Services, and
// EndpointSlices. Undns watches those objects and answers DNS queries. This
// package provides their shared names, labels, ConfigMap encoding, endpoint
// eligibility rules, and retirement timing; it does not perform I/O.
//
// # Publication
//
// Krane writes [ConnectionData] with [Encode]; undns reads it with [Decode].
// An unresolved alias still has a positive revision but no deployment or Service:
//
//	data, err := privatenetwork.Encode(privatenetwork.ConnectionData{
//		Alias: "worker", Revision: 1,
//	})
//
// [ServiceName] identifies a deployment's shared discovery Service.
// [AppendReadyAddresses] keeps publisher readiness checks and DNS answers on
// the same endpoint eligibility rules.
package privatenetwork
