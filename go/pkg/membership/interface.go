package membership

import "github.com/unkeyed/unkey/go/pkg/discovery"

type Membership interface {
	Start(discovery.Discoverer) error
	Leave() error
	Members() ([]Member, error)
	SubscribeJoinEvents() <-chan Member

	SubscribeLeaveEvents() <-chan Member
}
