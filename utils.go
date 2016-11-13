package main

import (
	"errors"
	"net"

	"golang.org/x/net/icmp"
)

func resolveHost(c *icmp.PacketConn, protocol int, host string) (net.Addr, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	netaddr := func(ip net.IP) (net.Addr, error) {
		switch c.LocalAddr().(type) {
		case *net.UDPAddr:
			return &net.UDPAddr{IP: ip}, nil
		case *net.IPAddr:
			return &net.IPAddr{IP: ip}, nil
		default:
			return nil, errors.New("neither UDPAddr nor IPAddr")
		}
	}
	for _, ip := range ips {
		switch protocol {
		case ianaProtocolICMP:
			if ip.To4() != nil {
				return netaddr(ip)
			}
		case ianaProtocolIPv6ICMP:
			if ip.To16() != nil && ip.To4() == nil {
				return netaddr(ip)
			}
		}
	}
	return nil, errors.New("no A or AAAA record")
}
