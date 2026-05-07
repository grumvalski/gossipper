package hep

import (
	"errors"
	"net"
	"time"

	"github.com/qxip/gossipper/hepcodec"
	"github.com/qxip/gossipper/mediasink"
)

// Protocol type constants on the HEP wire (proto_type chunk).
const (
	ProtocolSIP        = hepcodec.ProtocolSIP
	ProtocolRTCP       = hepcodec.ProtocolRTCP
	ProtocolRTPReport  = hepcodec.ProtocolRTPReport
	ProtocolRTCPReport = hepcodec.ProtocolRTCPReport
	ProtocolDTMF       = hepcodec.ProtocolDTMF
)

// Message is a HEP message to encode.
type Message = hepcodec.Message

// Decoded is a parsed HEP message.
type Decoded = hepcodec.Decoded

// Encode encodes a HEP3 packet.
func Encode(msg Message) ([]byte, error) {
	return hepcodec.Encode(msg)
}

// Decode parses a HEP3 packet.
func Decode(packet []byte) (Decoded, error) {
	return hepcodec.Decode(packet)
}

// Config holds the HEP client configuration.
type Config struct {
	Addr            string
	CaptureID       uint32
	Password        string
	RawRTCP         bool
	HomerLakeRTCP   bool
	SendMediaReport bool
}

// Client is a HEP UDP client: SIP encoding and optional media export.
type Client struct {
	conn      *net.UDPConn
	addr      *net.UDPAddr
	captureID uint32
	password  string
	media     mediasink.MediaExporter
}

// New creates a new HEP client.
func New(cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		return nil, errors.New("hep addr is required")
	}
	addr, err := net.ResolveUDPAddr("udp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}
	media, err := mediasink.NewMediaExporter(conn, addr, mediasink.MediaConfig{
		CaptureID:       cfg.CaptureID,
		Password:        cfg.Password,
		RawRTCP:         cfg.RawRTCP,
		HomerLakeRTCP:   cfg.HomerLakeRTCP,
		SendMediaReport: cfg.SendMediaReport,
	})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Client{
		conn:      conn,
		addr:      addr,
		captureID: cfg.CaptureID,
		password:  cfg.Password,
		media:     media,
	}, nil
}

// Close shuts down media export goroutines and closes the UDP socket.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	if c.media != nil {
		_ = c.media.Close()
	}
	return c.conn.Close()
}

// SendSIP sends a SIP message via HEP.
func (c *Client) SendSIP(now time.Time, srcIP string, srcPort int, dstIP string, dstPort int, transport string, callID string, payload []byte) error {
	if c == nil {
		return nil
	}
	packet, err := hepcodec.Encode(hepcodec.Message{
		Time:          now,
		SrcIP:         srcIP,
		DstIP:         dstIP,
		SrcPort:       srcPort,
		DstPort:       dstPort,
		IPProtocol:    transportProtocol(transport),
		ProtoType:     ProtocolSIP,
		CaptureID:     c.captureID,
		AuthKey:       c.password,
		CorrelationID: callID,
		Payload:       payload,
	})
	if err != nil {
		return err
	}
	_, err = c.conn.WriteToUDP(packet, c.addr)
	return err
}

// SendRTP forwards RTP to the configured media exporter.
func (c *Client) SendRTP(now time.Time, srcIP string, srcPort int, dstIP string, dstPort int, callID string, payload []byte) error {
	if c == nil || c.media == nil {
		return nil
	}
	return c.media.SendRTP(now, srcIP, srcPort, dstIP, dstPort, callID, payload)
}

// SendRTCP forwards RTCP to the configured media exporter.
func (c *Client) SendRTCP(now time.Time, callID string, ssrc uint32, srcIP string, srcPort int, dstIP string, dstPort int, packetLoss uint32, rawPayload []byte) error {
	if c == nil || c.media == nil {
		return nil
	}
	return c.media.SendRTCP(now, callID, ssrc, srcIP, srcPort, dstIP, dstPort, packetLoss, rawPayload)
}

// SendFinalReports flushes end-of-call media reports when supported by the exporter.
func (c *Client) SendFinalReports(callID string) {
	if c == nil || c.media == nil {
		return
	}
	c.media.SendFinalReports(callID)
}

func transportProtocol(transport string) uint8 {
	switch transport {
	case "t1", "tn", "l1", "ln", "TCP", "tcp", "TLS", "tls":
		return hepcodec.IPProtoTCP
	default:
		return hepcodec.IPProtoUDP
	}
}
