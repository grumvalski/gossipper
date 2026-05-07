package mediasink

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/sipcapture/gossipper/hepcodec"
)

// streamPruneAge is how long a stream can be idle before being pruned.
const streamPruneAge = 60 * time.Second

type rtpStreamState struct {
	packetCount uint32
	octetCount  uint32
	lastSeen    time.Time
	reportStart time.Time

	srcIP   string
	srcPort int
	dstIP   string
	dstPort int
	callID  string

	lastRTPTimestamp uint32
	prevRTPTimestamp uint32
	lastSeq          uint16
	lastArrivalTime  time.Time
	firstPacket      bool
	mediaPT          uint8

	jitter     float64
	maxJitter  float64
	sumJitter  float64
	sumDelta   float64
	deltaCount uint32
	maxDelta   float64
	maxSkew    float64
	outOrder   uint32

	ntpMSW     uint32
	ntpLSW     uint32
	packetLoss uint32
}

type openExporter struct {
	conn             *net.UDPConn
	addr             *net.UDPAddr
	captureID        uint32
	password         string
	homerLake        bool
	rawPeriodic      bool
	immediateRawRTCP bool

	mu         sync.Mutex
	rtpStreams map[uint32]*rtpStreamState
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func newOpenExporter(conn *net.UDPConn, addr *net.UDPAddr, captureID uint32, password string, homerLake, rawPeriodic bool) *openExporter {
	e := &openExporter{
		conn:             conn,
		addr:             addr,
		captureID:        captureID,
		password:         password,
		homerLake:        homerLake,
		rawPeriodic:      rawPeriodic,
		immediateRawRTCP: rawPeriodic && !homerLake,
		rtpStreams:       make(map[uint32]*rtpStreamState),
		stopCh:           make(chan struct{}),
	}
	e.wg.Add(1)
	switch {
	case homerLake:
		go e.homerRTCPLoop()
	case rawPeriodic:
		go e.rawRTCPLoop()
	}
	return e
}

func (e *openExporter) Close() error {
	if e == nil || e.stopCh == nil {
		return nil
	}
	close(e.stopCh)
	e.wg.Wait()
	return nil
}

func (e *openExporter) SendRTP(now time.Time, srcIP string, srcPort int, dstIP string, dstPort int, callID string, payload []byte) error {
	if e == nil || len(payload) < 12 {
		return nil
	}

	pt := payload[1] & 0x7F
	seq := binary.BigEndian.Uint16(payload[2:4])
	rtpTS := binary.BigEndian.Uint32(payload[4:8])
	ssrc := binary.BigEndian.Uint32(payload[8:12])

	e.mu.Lock()
	state, ok := e.rtpStreams[ssrc]
	if !ok {
		state = &rtpStreamState{
			srcIP:       srcIP,
			srcPort:     srcPort,
			dstIP:       dstIP,
			dstPort:     dstPort,
			callID:      callID,
			reportStart: now,
		}
		e.rtpStreams[ssrc] = state
	} else if callID != "" {
		state.callID = callID
	}

	state.packetCount++
	state.octetCount += uint32(len(payload) - 12)

	if !state.lastArrivalTime.IsZero() {
		deltaSec := now.Sub(state.lastArrivalTime).Seconds()
		deltaMS := deltaSec * 1000

		state.sumDelta += deltaMS
		state.deltaCount++
		if deltaMS > state.maxDelta {
			state.maxDelta = deltaMS
		}

		clockRate := payloadTypeClockRate(pt)
		if clockRate == 0 {
			clockRate = 8000
		}
		transitDiff := deltaSec - float64(int32(rtpTS-state.prevRTPTimestamp))/float64(clockRate)
		if transitDiff < 0 {
			transitDiff = -transitDiff
		}
		d := transitDiff * 1000
		state.jitter += (d - state.jitter) / 16.0
		if state.jitter > state.maxJitter {
			state.maxJitter = state.jitter
		}
		state.sumJitter += state.jitter

		if !state.firstPacket && int16(seq-state.lastSeq) < 0 {
			state.outOrder++
		}
	} else {
		state.firstPacket = true
	}
	state.firstPacket = false

	state.prevRTPTimestamp = state.lastRTPTimestamp
	state.lastRTPTimestamp = rtpTS
	state.lastArrivalTime = now
	state.lastSeq = seq
	state.lastSeen = now

	if pt != 101 {
		state.mediaPT = pt
	}
	e.mu.Unlock()
	return nil
}

func (e *openExporter) SendRTCP(now time.Time, callID string, ssrc uint32, srcIP string, srcPort int, dstIP string, dstPort int, packetLoss uint32, rawPayload []byte) error {
	if e == nil {
		return nil
	}

	if e.immediateRawRTCP {
		packet, err := hepcodec.Encode(hepcodec.Message{
			Time:          now,
			SrcIP:         srcIP,
			DstIP:         dstIP,
			SrcPort:       srcPort,
			DstPort:       dstPort,
			IPProtocol:    hepcodec.IPProtoUDP,
			ProtoType:     hepcodec.ProtocolRTCP,
			CaptureID:     e.captureID,
			AuthKey:       e.password,
			CorrelationID: callID,
			Payload:       rawPayload,
		})
		if err != nil {
			return err
		}
		_, err = e.conn.WriteToUDP(packet, e.addr)
		return err
	}

	e.mu.Lock()
	state, ok := e.rtpStreams[ssrc]
	if !ok {
		state = &rtpStreamState{
			srcIP:   srcIP,
			srcPort: srcPort,
			dstIP:   dstIP,
			dstPort: dstPort,
		}
		e.rtpStreams[ssrc] = state
	}
	state.callID = callID
	state.packetLoss = packetLoss
	state.lastSeen = now
	if len(rawPayload) >= 20 {
		state.ntpMSW = binary.BigEndian.Uint32(rawPayload[8:12])
		state.ntpLSW = binary.BigEndian.Uint32(rawPayload[12:16])
	}
	e.mu.Unlock()
	return nil
}

func (e *openExporter) SendFinalReports(string) {}

func (e *openExporter) rawRTCPLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case now := <-ticker.C:
			e.sendRawRTCPReports(now)
		}
	}
}

func (e *openExporter) homerRTCPLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case now := <-ticker.C:
			e.sendHomerLakeRTCPReports(now)
		}
	}
}

func (e *openExporter) sendHomerLakeRTCPReports(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for ssrc, state := range e.rtpStreams {
		if now.Sub(state.lastSeen) > streamPruneAge {
			delete(e.rtpStreams, ssrc)
			continue
		}
		if state.packetCount == 0 {
			continue
		}
		payload, err := buildHomerLakeRTCPJSON(ssrc, state, now)
		if err != nil {
			continue
		}
		e.sendHEP(now, state.srcIP, state.srcPort, state.dstIP, state.dstPort, hepcodec.ProtocolRTCP, state.callID, payload)
	}
}

type homerLakeRTCPJSON struct {
	Type         int                        `json:"type"`
	Ssrc         int64                      `json:"ssrc"`
	ReportCount  int                        `json:"report_count"`
	ReportBlocks []homerLakeRTCPReportBlock `json:"report_blocks"`
	SenderInfo   homerLakeSenderInfo        `json:"sender_information"`
}

type homerLakeSenderInfo struct {
	Packets          int    `json:"packets"`
	NtpTimestampSec  string `json:"ntp_timestamp_sec"`
	NtpTimestampUsec string `json:"ntp_timestamp_usec"`
	RtpTimestamp     int64  `json:"rtp_timestamp"`
	Octets           int    `json:"octets"`
}

type homerLakeRTCPReportBlock struct {
	SourceSSRC   int64  `json:"source_ssrc"`
	FractionLost int    `json:"fraction_lost"`
	PacketsLost  int    `json:"packets_lost"`
	HighestSeqNo int    `json:"highest_seq_no"`
	Lsr          string `json:"lsr"`
	IaJitter     int    `json:"ia_jitter"`
	Dlsr         int    `json:"dlsr"`
}

func buildHomerLakeRTCPJSON(ssrc uint32, state *rtpStreamState, now time.Time) ([]byte, error) {
	ntpMSW := state.ntpMSW
	ntpLSW := state.ntpLSW
	if ntpMSW == 0 && ntpLSW == 0 {
		const ntpEpochOffset = 2208988800
		ntpMSW = uint32(now.Unix()) + ntpEpochOffset
		ntpLSW = uint32(float64(now.Nanosecond()) / 1e9 * (1 << 32))
	}

	msg := homerLakeRTCPJSON{
		Type:         200,
		Ssrc:         int64(ssrc),
		ReportCount:  0,
		ReportBlocks: []homerLakeRTCPReportBlock{},
		SenderInfo: homerLakeSenderInfo{
			Packets:          int(state.packetCount),
			NtpTimestampSec:  strconv.FormatUint(uint64(ntpMSW), 10),
			NtpTimestampUsec: strconv.FormatUint(uint64(ntpLSW), 10),
			RtpTimestamp:     int64(int32(state.lastRTPTimestamp)),
			Octets:           int(state.octetCount),
		},
	}
	return json.Marshal(msg)
}

func (e *openExporter) sendRawRTCPReports(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for ssrc, state := range e.rtpStreams {
		if now.Sub(state.lastSeen) > streamPruneAge {
			delete(e.rtpStreams, ssrc)
			continue
		}
		if state.packetCount == 0 {
			continue
		}
		sr := buildRTCPSR(ssrc, state, now)
		e.sendHEP(now, state.srcIP, state.srcPort, state.dstIP, state.dstPort, hepcodec.ProtocolRTCP, state.callID, sr)
	}
}

func buildRTCPSR(ssrc uint32, state *rtpStreamState, now time.Time) []byte {
	sr := make([]byte, 28)
	sr[0] = 0x80
	sr[1] = 200
	binary.BigEndian.PutUint16(sr[2:4], 6)
	binary.BigEndian.PutUint32(sr[4:8], ssrc)

	ntpMSW := state.ntpMSW
	ntpLSW := state.ntpLSW
	if ntpMSW == 0 {
		const ntpEpochOffset = 2208988800
		ntpMSW = uint32(now.Unix()) + ntpEpochOffset
		ntpLSW = uint32(float64(now.Nanosecond()) / 1e9 * (1 << 32))
	}
	binary.BigEndian.PutUint32(sr[8:12], ntpMSW)
	binary.BigEndian.PutUint32(sr[12:16], ntpLSW)

	binary.BigEndian.PutUint32(sr[16:20], uint32(now.UnixNano()/125))
	binary.BigEndian.PutUint32(sr[20:24], state.packetCount)
	binary.BigEndian.PutUint32(sr[24:28], state.octetCount)
	return sr
}

func (e *openExporter) sendHEP(now time.Time, srcIP string, srcPort int, dstIP string, dstPort int, protoType uint8, correlationID string, payload []byte) {
	packet, err := hepcodec.Encode(hepcodec.Message{
		Time:          now,
		SrcIP:         srcIP,
		DstIP:         dstIP,
		SrcPort:       srcPort,
		DstPort:       dstPort,
		IPProtocol:    hepcodec.IPProtoUDP,
		ProtoType:     protoType,
		CaptureID:     e.captureID,
		AuthKey:       e.password,
		CorrelationID: correlationID,
		Payload:       payload,
	})
	if err != nil {
		return
	}
	_, _ = e.conn.WriteToUDP(packet, e.addr)
}

func payloadTypeClockRate(pt uint8) uint32 {
	switch pt {
	case 0, 3, 8, 9, 18:
		return 8000
	case 96, 97, 98:
		return 90000
	default:
		return 8000
	}
}

var _ MediaExporter = (*openExporter)(nil)
