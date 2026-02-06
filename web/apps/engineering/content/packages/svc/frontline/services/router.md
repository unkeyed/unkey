---
title: router
description: "handles routing lookups and sentinel selection for the frontline"
---

Package router handles routing lookups and sentinel selection for the frontline.

The router service is responsible for:

  - Looking up frontline routes by hostname
  - Finding available sentinels for an environment
  - Selecting the best sentinel based on region proximity and health

### Routing Strategy

The router uses a simple and efficient strategy:

  - If a healthy sentinel exists in the local region, route to it directly
  - If no local sentinel, route to the nearest region's NLB

### Example Flow

Request to hostname in us-east-1.aws, received in eu-west-1.aws (no local sentinel):

 1. eu-west-1.aws frontline receives request
 2. Lookup shows environment has sentinels in us-east-1.aws, ap-south-1.aws
 3. eu-west-1.aws has no local sentinel
 4. Select nearest region with healthy sentinel (us-east-1.aws)
 5. Forward to us-east-1.aws NLB
 6. us-east-1.aws frontline routes to local sentinel

## Variables

regionProximity maps regions to their closest regions in order of proximity. Format: region.cloud (e.g., "us-east-1.aws")
```go
var regionProximity = map[string][]string{

	"us-east-1.aws": {"us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ca-central-1.aws", "eu-west-1.aws", "eu-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws"},
	"us-east-2.aws": {"us-east-1.aws", "us-west-2.aws", "us-west-1.aws", "ca-central-1.aws", "eu-west-1.aws", "eu-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws"},

	"us-west-1.aws": {"us-west-2.aws", "us-east-2.aws", "us-east-1.aws", "ca-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws", "eu-west-1.aws", "eu-central-1.aws"},
	"us-west-2.aws": {"us-west-1.aws", "us-east-2.aws", "us-east-1.aws", "ca-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws", "eu-west-1.aws", "eu-central-1.aws"},

	"ca-central-1.aws": {"us-east-2.aws", "us-east-1.aws", "us-west-2.aws", "us-west-1.aws", "eu-west-1.aws", "eu-central-1.aws", "ap-northeast-1.aws", "ap-southeast-1.aws", "ap-southeast-2.aws"},

	"eu-west-1.aws":    {"eu-west-2.aws", "eu-central-1.aws", "eu-north-1.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},
	"eu-west-2.aws":    {"eu-west-1.aws", "eu-central-1.aws", "eu-north-1.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},
	"eu-central-1.aws": {"eu-west-1.aws", "eu-west-2.aws", "eu-north-1.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},
	"eu-north-1.aws":   {"eu-central-1.aws", "eu-west-1.aws", "eu-west-2.aws", "us-east-1.aws", "us-east-2.aws", "us-west-2.aws", "us-west-1.aws", "ap-south-1.aws", "ap-southeast-1.aws"},

	"ap-south-1.aws":     {"ap-southeast-1.aws", "ap-southeast-2.aws", "ap-northeast-1.aws", "eu-west-1.aws", "eu-central-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws"},
	"ap-northeast-1.aws": {"ap-southeast-1.aws", "ap-southeast-2.aws", "ap-south-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws", "eu-west-1.aws", "eu-central-1.aws"},
	"ap-southeast-1.aws": {"ap-southeast-2.aws", "ap-northeast-1.aws", "ap-south-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws", "eu-west-1.aws", "eu-central-1.aws"},
	"ap-southeast-2.aws": {"ap-southeast-1.aws", "ap-northeast-1.aws", "ap-south-1.aws", "us-west-2.aws", "us-west-1.aws", "us-east-1.aws", "us-east-2.aws", "eu-west-1.aws", "eu-central-1.aws"},

	"local.dev": {},
}
```


## Types

### type Config

```go
type Config struct {
	Region                 string
	DB                     db.Database
	FrontlineRouteCache    cache.Cache[string, db.FrontlineRoute]
	SentinelsByEnvironment cache.Cache[string, []db.Sentinel]
}
```

### type RouteDecision

```go
type RouteDecision struct {
	// LocalSentinel is set if there's a healthy sentinel in the local region
	LocalSentinel *db.Sentinel

	// NearestNLBRegion is set if we need to forward to another region's NLB
	NearestNLBRegion string

	// DeploymentID to pass in X-Unkey-Deployment-Id header
	DeploymentID string
}
```

### type Service

```go
type Service interface {
	LookupByHostname(ctx context.Context, hostname string) (*db.FrontlineRoute, []db.Sentinel, error)
	SelectSentinel(route *db.FrontlineRoute, sentinels []db.Sentinel) (*RouteDecision, error)
}
```

