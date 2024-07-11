package membership

type Membership interface {
	Join(addrs ...string) (int, error)
	Leave() error
	Members() ([]Member, error)
	SerfAddr() string
	SubscribeJoinEvents() <-chan Member

	SubscribeLeaveEvents() <-chan Member

	NodeId() string
}
