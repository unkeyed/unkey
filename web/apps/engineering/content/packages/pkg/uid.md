---
title: uid
description: "generates prefixed random identifiers for Unkey resources"
---

Package uid generates prefixed random identifiers for Unkey resources.

The package provides two main functions for generating random strings: \[New] for prefixed identifiers and \[DNS1035] for DNS-compliant labels.

### Security

This package uses math/rand/v2 which is NOT cryptographically secure. The generated identifiers are predictable and MUST NOT be used for API keys, session tokens, or any security-sensitive purposes. Use crypto/rand directly for those cases.

### Usage

Generate a prefixed identifier:

	id := uid.New(uid.KeyPrefix)      // "key_k3n5p8x2"
	id := uid.New(uid.APIPrefix, 12)  // "api_a9k2n5p8x3m7"

Generate a DNS-1035 compliant label:

	label := uid.DNS1035()    // "k3n5p8x2" (starts with letter)
	label := uid.DNS1035(12)  // "a9k2n5p8x3m7"

### Prefixes

Standard prefixes are defined as \[Prefix] constants (KeyPrefix, APIPrefix, WorkspacePrefix, etc.) to make IDs self-descriptive. See prefix.go for the complete list.

## Constants

```go
const (
	dns1035Alpha    = "abcdefghijklmnopqrstuvwxyz"
	dns1035AlphaNum = dns1035Alpha + "0123456789"
)
```

```go
const defaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
```


## Functions

### func DNS1035

```go
func DNS1035(length ...int) string
```

DNS1035 generates a random string compliant with RFC 1035 DNS label rules.

The first character is always a lowercase letter; subsequent characters are lowercase letters or digits. Default length is 8 characters; pass a custom length to override.

Uses math/rand/v2 which is NOT cryptographically secure.

### func New

```go
func New(prefix Prefix, length ...int) string
```

New generates a prefixed random identifier.

The identifier consists of the prefix, an underscore separator, and random alphanumeric characters. Default random portion is 8 characters; pass a custom length to override.

Pass an empty prefix to generate an identifier without a prefix.

Uses math/rand/v2 which is NOT cryptographically secure. Do not use for API keys, tokens, or security-sensitive purposes.

### func Secure

```go
func Secure(length ...int) string
```

Secure generates a cryptographically secure random identifier.

Uses crypto/rand for secure random generation. Use this for verification tokens, API keys, or any security-sensitive purposes.

The identifier consists of random alphanumeric characters. Default length is 24 characters; pass a custom length to override.


## Types

### type Prefix

```go
type Prefix string
```

Prefix is a resource type identifier prepended to generated IDs.

```go
const (
	KeyPrefix                 Prefix = "key"
	APIPrefix                 Prefix = "api"
	RequestPrefix             Prefix = "req"
	WorkspacePrefix           Prefix = "ws"
	KeySpacePrefix            Prefix = "ks" // keyspace
	RolePrefix                Prefix = "role"
	TestPrefix                Prefix = "test" // for tests only
	RatelimitNamespacePrefix  Prefix = "rlns"
	RatelimitOverridePrefix   Prefix = "rlor"
	PermissionPrefix          Prefix = "perm"
	IdentityPrefix            Prefix = "id"
	RatelimitPrefix           Prefix = "rl"
	AuditLogPrefix            Prefix = "log"
	InstancePrefix            Prefix = "ins"
	SentinelPrefix            Prefix = "se"
	CiliumNetworkPolicyPrefix Prefix = "net"
	OrgPrefix                 Prefix = "org"

	// Control plane prefixes
	ProjectPrefix        Prefix = "proj"
	EnvironmentPrefix    Prefix = "env"
	DomainPrefix         Prefix = "dom"
	DeploymentPrefix     Prefix = "d"
	FrontlineRoutePrefix Prefix = "flr"
	CertificatePrefix    Prefix = "cert"
)
```

