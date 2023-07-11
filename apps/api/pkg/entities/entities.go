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
	Remaining      struct {
		// Whether or not the value in `Remaining` makes any sense or is just a default
		Enabled   bool
		Remaining int64
	}
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
