package launcher

import (
	"fmt"

	"github.com/sipcapture/gossipper/internal/cli"
	"github.com/sipcapture/gossipper/internal/scenario"
)

// ValidateScenario checks transport, injection, and 3PCC rules for a parsed scenario
// using the same rules as Prepare, without building engine config.
func ValidateScenario(cfg cli.Config, sc scenario.Scenario) error {
	cfgCopy := cfg
	if err := NormalizeTransport(&cfgCopy, sc); err != nil {
		return err
	}
	if cfgCopy.InjectionFile != "" && sc.Mode == scenario.ModeServer && cfgCopy.Transport != "ui" {
		return fmt.Errorf("injection (-inf / -ip_field) is only supported for server transport ui")
	}
	return Validate3PCCRole(cfgCopy, sc)
}
