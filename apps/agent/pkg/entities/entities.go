package entities

import "time"

type Key struct {
	Id             string
	KeyAuthId      string
	WorkspaceId    string
	Name           string
	Hash           string
	Start          string
	OwnerId        string
	Meta           map[string]any
	CreatedAt      time.Time
	Expires        time.Time
	Ratelimit      *Ratelimit
	ForWorkspaceId string
	// How many more times this key may be verified
	// nil == disabled
	Remaining *int32
}

type Ratelimit struct {
	Type           string
	Limit          int32
	RefillRate     int32
	RefillInterval int32
}

type AuthType string

const (
	AuthTypeKey AuthType = "key"
	AuthTypeJWT AuthType = "jwt"
)

type Api struct {
	Id          string
	Name        string
	WorkspaceId string
	IpWhitelist []string

	AuthType AuthType
	// Only set if AuthType == "key"
	KeyAuthId string
}

type Plan string

const (
	FreePlan       Plan = "free"
	ProPlan        Plan = "pro"
	EnterprisePlan Plan = "enterprise"
)

type Workspace struct {
	Id                 string
	Name               string
	TenantId           string
	Internal           bool
	EnableBetaFeatures bool
	Plan               Plan
}

type KeyAuth struct {
	Id          string
	WorkspaceId string
}
