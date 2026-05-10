package launcher

import (
	"testing"

	"github.com/sipcapture/gossipper/internal/cli"
	"github.com/sipcapture/gossipper/internal/scenario"
)

func TestValidateScenarioInjectionRule(t *testing.T) {
	sc := scenario.Scenario{Mode: scenario.ModeServer}
	cfg := cli.DefaultConfig()
	cfg.Transport = "u1"
	cfg.InjectionFile = "/tmp/x.csv"
	if err := ValidateScenario(cfg, sc); err == nil {
		t.Fatal("expected error for server + injection without ui transport")
	}
	cfg.Transport = "ui"
	cfg.UISourceIPs = []string{"10.0.0.1"}
	cfg.IPField = 0
	if err := ValidateScenario(cfg, sc); err != nil {
		t.Fatal(err)
	}
}
