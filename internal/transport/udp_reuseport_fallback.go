//go:build !(linux || darwin || freebsd || openbsd || netbsd)

package transport

import "net"

func listenParallelUDP(addr *net.UDPAddr, receivers int) ([]*net.UDPConn, error) {
	_, _ = receivers, receivers
	c, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	tuneUDPConn(c)
	return []*net.UDPConn{c}, nil
}

func maybeSingleReceiverOS(n int) int { return 1 }
