package regn

import (
	"net"
	"sync"
)

type domain struct {
	ipv4       net.IP
	ipv4String string
	ipv6       net.IP
	ipv6String string
}

var dns map[string]*domain = make(map[string]*domain)
var dnsLock sync.RWMutex

func HostToIp(host string, ipv6 bool) (net.IP, error) {
	var dn *domain = &domain{}
	dnsLock.Lock()
	dn, ok := dns[host]
	if !ok {
		dn = &domain{}
		ips, err := net.LookupIP(host)
		if err != nil {
			dnsLock.Unlock()
			return nil, err
		}

		for _, ipv := range ips {
			if v4 := ipv.To4(); v4 != nil {
				dn.ipv4 = v4
				dn.ipv4String = v4.String()
			} else if v6 := ipv.To16(); v6 != nil {
				dn.ipv6 = v6
				dn.ipv6String = v6.String()
			}
		}
		dns[host] = dn
	}

	if ipv6 && dn.ipv6 != nil {
		dnsLock.Unlock()
		return dn.ipv6, nil
	}

	dnsLock.Unlock()
	return dn.ipv4, nil
}
