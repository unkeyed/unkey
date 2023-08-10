package whitelist

import "net"

// Ip checks if the sourceIp and returns `true` if it is whitelisted
func Ip(sourceIp string, whitelisted []string) bool {
	s := net.ParseIP(sourceIp)
	if s == nil {
		return false
	}

	for _, w := range whitelisted {
		if s.Equal(net.ParseIP(w)) {
			return true
		}
	}

	return false

}
