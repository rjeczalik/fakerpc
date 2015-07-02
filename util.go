package fakerpc

import (
	"errors"
	"net"
	"net/url"
	"strconv"
)

func ipnil(ip net.IP) net.IP {
	if ip == nil {
		return net.IPv4(0, 0, 0, 0)
	}
	return ip
}

func iptomask(ip net.IP) net.IPMask {
	ip = ipnil(ip)
	return net.IPv4Mask(ip[12], ip[13], ip[14], ip[15])
}

func masktoip(mask net.IPMask) (ip net.IP) {
	if mask != nil {
		ip = net.IPv4(mask[0], mask[1], mask[2], mask[3])
	}
	return ipnil(ip)
}

var addrcache = make(map[string]*net.TCPAddr)

func tcpaddr(addr net.Addr) (*net.TCPAddr, error) {
	tcpa, ok := addr.(*net.TCPAddr)
	if ok {
		return tcpa, nil
	}
	tcpa, ok = addrcache[addr.String()]
	if ok {
		return tcpa, nil
	}
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, err
	}
	tcpa = &net.TCPAddr{}
	if tcpa.Port, err = strconv.Atoi(port); err != nil {
		return nil, err
	}
	if tcpa.IP = net.ParseIP(host); tcpa.IP != nil {
		addrcache[addr.String()] = tcpa
		return tcpa, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	tcpa.IP = ips[0]
	addrcache[addr.String()] = tcpa
	return tcpa, nil
}

func tcpaddrnil(addr net.Addr) (tcpa *net.TCPAddr) {
	if a, err := tcpaddr(addr); err == nil {
		tcpa = a
	}
	return
}

func tcpaddrequal(lhs, rhs *net.TCPAddr) bool {
	return lhs == rhs || (lhs.IP.Equal(rhs.IP) && lhs.Port == rhs.Port)
}

func ipnetaddr(addr net.Addr) ([]*net.IPNet, error) {
	ip, err := tcpaddr(addr)
	if err != nil {
		return nil, err
	}
	ifi, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var all []*net.IPNet
	for _, ifi := range ifi {
		addr, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addr {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.Contains(ip.IP) {
					return []*net.IPNet{ipnet}, nil
				}
				all = append(all, ipnet)
			}
		}
	}
	if len(all) == 0 {
		return nil, errors.New("fakerpc: unable to find single network address for " + addr.String())
	}
	return all, nil
}

type hpwrap string

func (w hpwrap) Network() string { return string(w) }
func (w hpwrap) String() string  { return string(w) }

func urltotcpaddr(u *url.URL) (*net.TCPAddr, error) {
	hp := u.Host
	if _, _, err := net.SplitHostPort(hp); err != nil {
		hp = hp + ":80"
	}
	return tcpaddr(hpwrap(hp))
}
