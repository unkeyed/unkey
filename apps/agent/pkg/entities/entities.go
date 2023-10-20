package entities

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
	Id       string
	Name     string
	TenantId string
	Plan     Plan
}

type KeyAuth struct {
	Id          string
	WorkspaceId string
}
