package rate

import (
	"net"
)

type IP = string

func ip(addr net.Addr) IP {
	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}
	return ip
}
