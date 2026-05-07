// Package mediasink defines the public contract for RTP/RTCP HEP export used by gossipper.
//
// Open-source gossipper implements the default paths in open.go (raw RTCP SR and Homer-Lake JSON).
// Optional short JSON media reporting (HEP types 0x22 / 0x24 / 0x64) can be linked at compile time
// via RegisterMediaExporterExtension from a separate module.
package mediasink

import (
	"errors"
	"net"
	"time"
)

// ErrShortJSONMediaUnavailable is returned when short JSON media reporting is requested
// (-send_media_report with neither raw RTCP nor Homer-Lake) and no extension factory is registered.
var ErrShortJSONMediaUnavailable = errors.New("mediasink: short JSON media reports require a registered extension (or use -hep_raw_rtcp=true / -hep_homer_lake_rtcp=true)")

// MediaExporter mirrors RTP/RTCP into HEP and flushes on call end.
type MediaExporter interface {
	SendRTP(now time.Time, srcIP string, srcPort int, dstIP string, dstPort int, callID string, payload []byte) error
	SendRTCP(now time.Time, callID string, ssrc uint32, srcIP string, srcPort int, dstIP string, dstPort int, packetLoss uint32, rawPayload []byte) error
	SendFinalReports(callID string)
	Close() error
}

// MediaConfig selects which media export path is active.
type MediaConfig struct {
	CaptureID       uint32
	Password        string
	RawRTCP         bool // CLI -hep_raw_rtcp (ignored when HomerLakeRTCP is true)
	HomerLakeRTCP   bool
	SendMediaReport bool
}

// MediaFactory builds a MediaExporter for a resolved HEP UDP session.
type MediaFactory func(conn *net.UDPConn, addr *net.UDPAddr, cfg MediaConfig) (MediaExporter, error)
