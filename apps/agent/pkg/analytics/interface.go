package analytics

import "context"

type Analytics interface {
	PublishKeyVerificationEvent(ctx context.Context, event KeyVerificationEvent)
	GetKeyStats(ctx context.Context, keyId string) (KeyStats, error)
}

type KeyVerificationEvent struct {
	ApiId       string `json:"apiId"`
	WorkspaceId string `json:"workspaceId"`
	KeyId       string `json:"keyId"`
	Ratelimited bool   `json:"ratelimited"`
	Time        int64  `json:"time"`
}

type ValueAtTime struct {
	Time  int64
	Value int64
}
type KeyStats struct {
	Usage []ValueAtTime
}
