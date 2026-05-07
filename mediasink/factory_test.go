package mediasink

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

type fakeExporter struct {
	mu           sync.Mutex
	rtpCalls     int
	rtcpCalls    int
	finalCalls   int
	closeCalls   int
	lastCallID   string
	failSendRTP  bool
	failSendRTCP bool
}

func (f *fakeExporter) SendRTP(now time.Time, srcIP string, srcPort int, dstIP string, dstPort int, callID string, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rtpCalls++
	f.lastCallID = callID
	if f.failSendRTP {
		return errors.New("fake rtp error")
	}
	return nil
}

func (f *fakeExporter) SendRTCP(now time.Time, callID string, ssrc uint32, srcIP string, srcPort int, dstIP string, dstPort int, packetLoss uint32, rawPayload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rtcpCalls++
	f.lastCallID = callID
	if f.failSendRTCP {
		return errors.New("fake rtcp error")
	}
	return nil
}

func (f *fakeExporter) SendFinalReports(callID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finalCalls++
	f.lastCallID = callID
}

func (f *fakeExporter) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closeCalls++
	return nil
}

func TestNewMediaExporterDisabledReturnsNil(t *testing.T) {
	t.Parallel()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	uc := pc.(*net.UDPConn)
	addr := uc.LocalAddr().(*net.UDPAddr)

	ex, err := NewMediaExporter(uc, addr, MediaConfig{SendMediaReport: false})
	if err != nil {
		t.Fatalf("NewMediaExporter: %v", err)
	}
	if ex != nil {
		t.Fatal("expected nil exporter when SendMediaReport is false")
	}
}

func TestNewMediaExporterShortJSONWithoutExtensionFails(t *testing.T) {
	t.Parallel()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	uc := pc.(*net.UDPConn)
	addr := uc.LocalAddr().(*net.UDPAddr)

	_, err = NewMediaExporter(uc, addr, MediaConfig{
		SendMediaReport: true,
		RawRTCP:         false,
		HomerLakeRTCP:   false,
	})
	if !errors.Is(err, ErrShortJSONMediaUnavailable) {
		t.Fatalf("expected ErrShortJSONMediaUnavailable, got %v", err)
	}
}

func TestNewMediaExporterShortJSONWithExtension(t *testing.T) {
	RegisterMediaExporterExtension(func(conn *net.UDPConn, addr *net.UDPAddr, cfg MediaConfig) (MediaExporter, error) {
		if !cfg.SendMediaReport {
			return nil, nil
		}
		rawPeriodic := cfg.RawRTCP && !cfg.HomerLakeRTCP
		if cfg.HomerLakeRTCP || rawPeriodic {
			return nil, nil
		}
		return &fakeExporter{}, nil
	})
	defer RegisterMediaExporterExtension(nil)

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	uc := pc.(*net.UDPConn)
	addr := uc.LocalAddr().(*net.UDPAddr)

	ex, err := NewMediaExporter(uc, addr, MediaConfig{
		SendMediaReport: true,
		RawRTCP:         false,
		HomerLakeRTCP:   false,
	})
	if err != nil {
		t.Fatalf("NewMediaExporter: %v", err)
	}
	if ex == nil {
		t.Fatal("expected extension exporter")
	}
	_ = ex.Close()
}

func TestNewMediaExporterRawRTCPPeriodicOK(t *testing.T) {
	t.Parallel()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	uc := pc.(*net.UDPConn)
	addr := uc.LocalAddr().(*net.UDPAddr)

	ex, err := NewMediaExporter(uc, addr, MediaConfig{
		SendMediaReport: true,
		RawRTCP:         true,
		HomerLakeRTCP:   false,
	})
	if err != nil {
		t.Fatalf("NewMediaExporter: %v", err)
	}
	if ex == nil {
		t.Fatal("expected non-nil exporter")
	}
	if err := ex.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewMediaExporterHomerLakeOK(t *testing.T) {
	t.Parallel()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	uc := pc.(*net.UDPConn)
	addr := uc.LocalAddr().(*net.UDPAddr)

	ex, err := NewMediaExporter(uc, addr, MediaConfig{
		SendMediaReport: true,
		HomerLakeRTCP:   true,
		RawRTCP:         false,
	})
	if err != nil {
		t.Fatalf("NewMediaExporter: %v", err)
	}
	if ex == nil {
		t.Fatal("expected non-nil exporter")
	}
	if err := ex.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFakeExporterContract(t *testing.T) {
	t.Parallel()
	var f fakeExporter
	now := time.Now()
	_ = f.SendRTP(now, "10.0.0.1", 10000, "10.0.0.2", 20000, "cid", []byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	_ = f.SendRTCP(now, "cid", 1, "10.0.0.1", 10000, "10.0.0.2", 20000, 0, nil)
	f.SendFinalReports("bye")
	_ = f.Close()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rtpCalls != 1 || f.rtcpCalls != 1 || f.finalCalls != 1 || f.closeCalls != 1 {
		t.Fatalf("unexpected counts rtp=%d rtcp=%d final=%d close=%d", f.rtpCalls, f.rtcpCalls, f.finalCalls, f.closeCalls)
	}
}
