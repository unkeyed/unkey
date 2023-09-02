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

type KeyVerificationEvent struct {
	ApiId       string `json:"apiId"`
	WorkspaceId string `json:"workspaceId"`
	KeyId       string `json:"keyId"`
	Ratelimited bool   `json:"ratelimited"`
	UsageExceeded bool `json:"usageExceeded"`
	Time        int64  `json:"time"`

	EdgeRegion string `json:"edgeRegion"`
	Region     string `json:"region"`
	IpAddress  string `json:"ipAddress"`
	UserAgent  string `json:"userAgent"`

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
