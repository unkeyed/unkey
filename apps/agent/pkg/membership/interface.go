package membership

type Membership interface {
	Join(addrs ...string) (int, error)
	Leave() error
	Members() ([]Member, error)
	Addr() string
	SubscribeJoinEvents() <-chan Member

	SubscribeLeaveEvents() <-chan Member

	Sync() error
	NodeId() string
}

type Member struct {
	// Global unique identifier for the node
	Id      string `json:"id"`
	RpcAddr string `json:"addr"`
	State   string `json:"state"`
}
