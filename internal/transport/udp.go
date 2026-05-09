package transport

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
)

const (
	maxUDPDatagram = 65535
	// inboundChanDepth is the depth of SharedUDP Receive channel (was 128 before widening).
	inboundChanDepth = 512
	udpSocketRecvBuf = 8 * 1024 * 1024
	udpSocketSendBuf = 1024 * 1024
)

type Packet struct {
	Data []byte
	Addr *net.UDPAddr
}

// SharedUDP is a UDP socket shared by multiple logical SIP flows (gossipper UAC/UAS).
// Uses the standard library listener plus a read loop (same model as transport mode "un").
type SharedUDP struct {
	conn         *net.UDPConn
	incoming     chan Packet
	closeOnce    sync.Once
	incomingOnce sync.Once
	closed       atomic.Bool
}

func NewSharedUDP(localAddr string) (*SharedUDP, error) {
	addr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetReadBuffer(udpSocketRecvBuf)
	_ = conn.SetWriteBuffer(udpSocketSendBuf)

	s := &SharedUDP{
		conn:     conn,
		incoming: make(chan Packet, inboundChanDepth),
	}
	go s.readLoop()
	return s, nil
}

func (s *SharedUDP) readLoop() {
	buffer := make([]byte, maxUDPDatagram)
	for {
		n, addr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			s.shutdownIncoming()
			return
		}
		if n <= 0 || n > maxUDPDatagram {
			continue
		}
		if addr == nil {
			continue
		}
		payload := make([]byte, n)
		copy(payload, buffer[:n])
		p := Packet{Data: payload, Addr: addr}
		select {
		case s.incoming <- p:
		default:
			select {
			case s.incoming <- p:
			default:
			}
		}
	}
}

func (s *SharedUDP) shutdownIncoming() {
	s.incomingOnce.Do(func() {
		close(s.incoming)
	})
}

func (s *SharedUDP) Send(payload []byte, addr *net.UDPAddr) error {
	if s.closed.Load() {
		return net.ErrClosed
	}
	_, err := s.conn.WriteToUDP(payload, addr)
	return err
}

func (s *SharedUDP) Receive() <-chan Packet {
	return s.incoming
}

func (s *SharedUDP) LocalPort() int {
	if addr, ok := s.conn.LocalAddr().(*net.UDPAddr); ok && addr != nil {
		return addr.Port
	}
	return 0
}

func (s *SharedUDP) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		err = s.conn.Close()
		s.shutdownIncoming()
	})
	return err
}

type DialogUDP struct {
	conn   *net.UDPConn
	remote *net.UDPAddr
}

func NewDialogUDP(localAddr, remoteAddr string) (*DialogUDP, error) {
	local, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	remote, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", local)
	if err != nil {
		return nil, err
	}
	return &DialogUDP{conn: conn, remote: remote}, nil
}

func (d *DialogUDP) Send(payload []byte) error {
	_, err := d.conn.WriteToUDP(payload, d.remote)
	return err
}

func (d *DialogUDP) Receive(ctx context.Context) (Packet, error) {
	buffer := make([]byte, 65535)
	go func() {
		<-ctx.Done()
		_ = d.conn.SetReadDeadline(deadlineFromContext(ctx))
	}()
	n, addr, err := d.conn.ReadFromUDP(buffer)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return Packet{}, ctx.Err()
		}
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return Packet{}, ctx.Err()
		}
		return Packet{}, err
	}
	payload := make([]byte, n)
	copy(payload, buffer[:n])
	return Packet{Data: payload, Addr: addr}, nil
}

func (d *DialogUDP) LocalPort() int {
	if addr, ok := d.conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.Port
	}
	return 0
}

func (d *DialogUDP) Close() error {
	return d.conn.Close()
}
