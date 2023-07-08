package entities

import "time"

type Key struct {
	Id             string
	ApiId          string
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
}

type Ratelimit struct {
	Type           string
	Limit          int64
	RefillRate     int64
	RefillInterval int64
}

type Api struct {
	Id          string
	Name        string
	WorkspaceId string
	IpWhitelist []string
}

type Workspace struct {
	Id                 string
	Name               string
	Slug               string
	TenantId           string
	Internal           bool
	EnableBetaFeatures bool
}
