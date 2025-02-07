package membership

import "context"

type Membership interface {
	Join(ctx context.Context) (int, error)
	Leave(ctx context.Context) error
	Members(ctx context.Context) ([]Member, error)
	Addr() string
	SubscribeJoinEvents() <-chan Member

	SubscribeLeaveEvents() <-chan Member
}

type Member struct {
	// Global unique identifier for the node
	ID      string `json:"id"`
	RpcAddr string `json:"addr"`
}
