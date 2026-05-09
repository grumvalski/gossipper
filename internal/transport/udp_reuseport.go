//go:build linux || darwin || freebsd || openbsd || netbsd

package transport

import (
	"context"
	"errors"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func listenParallelUDP(addr *net.UDPAddr, receivers int) ([]*net.UDPConn, error) {
	if receivers <= 1 {
		c, err := net.ListenUDP("udp", addr)
		if err != nil {
			return nil, err
		}
		tuneUDPConn(c)
		return []*net.UDPConn{c}, nil
	}

	lc := net.ListenConfig{
		Control: reusePortControl,
	}
	out := make([]*net.UDPConn, 0, receivers)
	for i := 0; i < receivers; i++ {
		p, err := lc.ListenPacket(context.Background(), "udp", addr.String())
		if err != nil {
			for _, q := range out {
				_ = q.Close()
			}
			return nil, err
		}
		uc, ok := p.(*net.UDPConn)
		if !ok {
			_ = p.Close()
			for _, q := range out {
				_ = q.Close()
			}
			return nil, errors.New("transport: ListenPacket is not *net.UDPConn")
		}
		tuneUDPConn(uc)
		out = append(out, uc)
	}
	return out, nil
}

func reusePortControl(network, address string, c syscall.RawConn) error {
	var opErr error
	err := c.Control(func(fd uintptr) {
		if e := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); e != nil {
			opErr = e
			return
		}
		opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	})
	if err != nil {
		return err
	}
	return opErr
}

// maybeSingleReceiverOS is a no-op on this build: reuseport is available.
func maybeSingleReceiverOS(n int) int { return n }
