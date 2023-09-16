package analytics

import "context"

type Analytics interface {
	PublishKeyVerificationEvent(ctx context.Context, event KeyVerificationEvent)
	GetKeyStats(ctx context.Context, keyId string) (KeyStats, error)
}

type KeyVerificationV1Event struct {
	ApiId       string `json:"apiId"`
	WorkspaceId string `json:"workspaceId"`
	KeyId       string `json:"keyId"`
	Ratelimited bool   `json:"ratelimited"`
	Time        int64  `json:"time"`
}

type DeniedReason string

const (
	DeniedRateLimited   DeniedReason = "RATE_LIMITED"
	DeniedUsageExceeded DeniedReason = "USAGE_EXCEEDED"
)

type KeyVerificationEvent struct {
	ApiId       string `json:"apiId"`
	WorkspaceId string `json:"workspaceId"`
	KeyId       string `json:"keyId"`

	// Deprecated, use `Denied` instead
	Ratelimited bool `json:"ratelimited"`
	// Deprecated, use `Denied` instead
	UsageExceeded bool  `json:"usageExceeded"`
	Time          int64 `json:"time"`

	EdgeRegion string `json:"edgeRegion"`
	Region     string `json:"region"`
	IpAddress  string `json:"ipAddress"`
	UserAgent  string `json:"userAgent"`

	Denied DeniedReason `json:"deniedReason"`

	// custom field the user may provide, like a URL or some id
	RequestedResource string `json:"requestedResource"`
}

type ValueAtTime struct {
	Time  int64
	Value int64
}
type KeyStats struct {
	Usage []ValueAtTime
}
