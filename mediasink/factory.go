package mediasink

import (
	"net"
	"sync/atomic"
)

var extension atomic.Value // holds MediaFactory; nil when unset

// RegisterMediaExporterExtension registers an optional factory for short JSON media export.
// Pass nil to clear. It is consulted before ErrShortJSONMediaUnavailable when neither Homer-Lake nor raw periodic mode applies.
func RegisterMediaExporterExtension(f MediaFactory) {
	extension.Store(f)
}

func loadExtension() MediaFactory {
	v := extension.Load()
	if v == nil {
		return nil
	}
	f, _ := v.(MediaFactory)
	return f
}

// MediaExporterExtensionRegistered reports whether a non-nil extension factory has been registered.
func MediaExporterExtensionRegistered() bool {
	return loadExtension() != nil
}

// NewMediaExporter returns nil when SendMediaReport is false.
// When SendMediaReport is true but neither Homer-Lake nor raw periodic mode applies and no extension
// handles the config, returns ErrShortJSONMediaUnavailable.
func NewMediaExporter(conn *net.UDPConn, addr *net.UDPAddr, cfg MediaConfig) (MediaExporter, error) {
	if !cfg.SendMediaReport {
		return nil, nil
	}
	if ext := loadExtension(); ext != nil {
		ex, err := ext(conn, addr, cfg)
		if err != nil {
			return nil, err
		}
		if ex != nil {
			return ex, nil
		}
	}
	rawPeriodic := cfg.RawRTCP && !cfg.HomerLakeRTCP
	if !cfg.HomerLakeRTCP && !rawPeriodic {
		return nil, ErrShortJSONMediaUnavailable
	}
	return newOpenExporter(conn, addr, cfg.CaptureID, cfg.Password, cfg.HomerLakeRTCP, rawPeriodic), nil
}
