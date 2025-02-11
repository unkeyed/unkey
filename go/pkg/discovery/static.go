package discovery

type Static struct {
	Addrs []string
}

var _ Discoverer = (*Static)(nil)

func (s *Static) Discover() ([]string, error) {
	return s.Addrs, nil
}
