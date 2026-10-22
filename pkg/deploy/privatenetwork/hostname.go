package privatenetwork

// Zone is the private DNS zone for connections and a deployment's own replicas.
const Zone = "unkey.internal"

// SelfName is the reserved connection name that resolves a deployment's own
// ready replicas across regions. Every deployment uses the same name; DNS
// scopes it to the querying deployment.
const SelfName = "self"

// SelfHost is [SelfName] in [Zone].
const SelfHost = SelfName + "." + Zone
