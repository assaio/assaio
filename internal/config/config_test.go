package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Since != "30d" || c.Format != "table" {
		t.Fatalf("defaults wrong: %+v", c)
	}
	if !c.Privacy.Anonymize {
		t.Fatalf("defaults wrong: Privacy.Anonymize = false, want true")
	}
}

func TestPrivacyFromYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte("privacy:\n  anonymize: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Privacy.Anonymize {
		t.Fatalf("Privacy.Anonymize = true, want false from file override")
	}
}

func TestFileThenEnv(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte("since: 7d\nformat: json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASSAIO_FORMAT", "csv")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Since != "7d" || c.Format != "csv" { // env overrides file
		t.Fatalf("merge wrong: %+v", c)
	}
}

func TestLoadPluginsFromYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	body := "plugins:\n  - name: mytool\n    command: /path/to/assaio-parser-mytool\n    timeout: 45s\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Plugins) != 1 {
		t.Fatalf("Plugins = %+v, want 1 entry", c.Plugins)
	}
	got := c.Plugins[0]
	if got.Name != "mytool" || got.Command != "/path/to/assaio-parser-mytool" || got.Timeout != "45s" {
		t.Fatalf("Plugins[0] = %+v", got)
	}
}

func TestSourcesEmptyByDefault(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Sources.Claude) != 0 || len(c.Sources.Codex) != 0 || len(c.Sources.Gemini) != 0 || len(c.Sources.Cline) != 0 {
		t.Fatalf("Sources = %+v, want all empty by default", c.Sources)
	}
}

func TestSourcesFromYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	body := "sources:\n  claude:\n    - /custom/claude/logs\n    - /other/claude/logs\n  codex:\n    - /custom/codex\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	wantClaude := []string{"/custom/claude/logs", "/other/claude/logs"}
	if len(c.Sources.Claude) != len(wantClaude) || c.Sources.Claude[0] != wantClaude[0] || c.Sources.Claude[1] != wantClaude[1] {
		t.Fatalf("Sources.Claude = %v, want %v", c.Sources.Claude, wantClaude)
	}
	if len(c.Sources.Codex) != 1 || c.Sources.Codex[0] != "/custom/codex" {
		t.Fatalf("Sources.Codex = %v, want [/custom/codex]", c.Sources.Codex)
	}
	if len(c.Sources.Gemini) != 0 || len(c.Sources.Cline) != 0 {
		t.Fatalf("unconfigured tools must stay empty: Sources = %+v", c.Sources)
	}
}

func TestSourcesFromEnv(t *testing.T) {
	t.Setenv("ASSAIO_SOURCES_CLAUDE", "/env/claude/logs")
	c, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Sources.Claude) != 1 || c.Sources.Claude[0] != "/env/claude/logs" {
		t.Fatalf("Sources.Claude = %v, want a single-element [/env/claude/logs] (env vars carry one root, not a comma-split list)", c.Sources.Claude)
	}
}

func TestServerFromYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	body := "server:\n  addr: \":9000\"\n  token: sekret\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Addr != ":9000" || c.Server.Token != "sekret" {
		t.Fatalf("Server = %+v, want {Addr::9000 Token:sekret}", c.Server)
	}
}

func TestSyncFromYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	body := "sync:\n  server: http://localhost:8787\n  token: sekret\n  member: alice\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sync.Server != "http://localhost:8787" || c.Sync.Token != "sekret" || c.Sync.Member != "alice" {
		t.Fatalf("Sync = %+v, want full struct populated from YAML", c.Sync)
	}
}

func TestServerTokenFromEnv(t *testing.T) {
	t.Setenv("ASSAIO_SERVER_TOKEN", "env-token")
	c, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Server.Token != "env-token" {
		t.Fatalf("Server.Token = %q, want env-token", c.Server.Token)
	}
}

func TestSyncEmptyByDefault(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Sync.Member != "" || c.Sync.Server != "" || c.Sync.Token != "" {
		t.Fatalf("Sync = %+v, want all empty by default", c.Sync)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		c       Config
		wantErr bool
	}{
		{"valid table", Config{Format: "table", Since: "30d"}, false},
		{"valid json", Config{Format: "json", Since: "7d"}, false},
		{"valid csv zero days", Config{Format: "csv", Since: "0d"}, false},
		{"invalid format", Config{Format: "xml", Since: "30d"}, true},
		{"invalid since suffix", Config{Format: "table", Since: "30days"}, true},
		{"invalid since negative", Config{Format: "table", Since: "-1d"}, true},
		{"invalid since empty", Config{Format: "table", Since: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePlugins(t *testing.T) {
	base := Config{Format: "table", Since: "30d"}
	tests := []struct {
		name    string
		p       PluginConfig
		wantErr bool
	}{
		{"valid full", PluginConfig{Name: "mytool", Command: "/usr/local/bin/x", Timeout: "60s"}, false},
		{"valid no timeout", PluginConfig{Name: "mytool", Command: "/usr/local/bin/x"}, false},
		{"invalid name uppercase", PluginConfig{Name: "MyTool", Command: "/usr/local/bin/x"}, true},
		{"invalid name spaces", PluginConfig{Name: "my tool", Command: "/usr/local/bin/x"}, true},
		{"empty name", PluginConfig{Name: "", Command: "/usr/local/bin/x"}, true},
		{"empty command", PluginConfig{Name: "mytool", Command: ""}, true},
		{"invalid timeout", PluginConfig{Name: "mytool", Command: "/usr/local/bin/x", Timeout: "not-a-duration"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := base
			c.Plugins = []PluginConfig{tt.p}
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPricingUnconfiguredByDefault(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Pricing.Configured() || c.Pricing.IsSubscription() {
		t.Fatalf("Pricing = %+v, want unconfigured API basis by default", c.Pricing)
	}
}

func TestPricingFromYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	body := "pricing:\n  mode: subscription\n  effective_per_token: 0.0000008\n  monthly_subscription_cost: 200\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Pricing.IsSubscription() {
		t.Fatalf("Pricing.Mode = %q, want subscription", c.Pricing.Mode)
	}
	if !approxEqual(c.Pricing.EffectivePerToken, 0.0000008) {
		t.Fatalf("EffectivePerToken = %g, want ~0.0000008", c.Pricing.EffectivePerToken)
	}
	if c.Pricing.MonthlySubscriptionCost != 200 {
		t.Fatalf("MonthlySubscriptionCost = %g, want 200", c.Pricing.MonthlySubscriptionCost)
	}
	cost, ok := c.Pricing.EffectiveWindowCost(1_000_000)
	if !ok || !approxEqual(cost, 0.8) {
		t.Fatalf("EffectiveWindowCost(1e6) = %v, %v; want ~0.8, true", cost, ok)
	}
}

// approxEqual compares two floats within a tolerance, since decimal config values like
// 0.0000008 are not exactly representable and repricing multiplies the rounding error.
func approxEqual(a, b float64) bool {
	const eps = 1e-12
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func TestPricingModeFromEnv(t *testing.T) {
	t.Setenv("ASSAIO_PRICING_MODE", "subscription")
	c, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Pricing.IsSubscription() {
		t.Fatalf("Pricing.Mode = %q, want subscription from env", c.Pricing.Mode)
	}
}

func TestPricingValidate(t *testing.T) {
	base := Config{Format: "table", Since: "30d"}
	tests := []struct {
		name    string
		p       Pricing
		wantErr bool
	}{
		{"empty ok", Pricing{}, false},
		{"api ok", Pricing{Mode: "api"}, false},
		{"subscription ok", Pricing{Mode: "subscription", EffectivePerToken: 1e-6, MonthlySubscriptionCost: 200}, false},
		{"bad mode", Pricing{Mode: "flat"}, true},
		{"negative rate", Pricing{EffectivePerToken: -1}, true},
		{"negative monthly", Pricing{MonthlySubscriptionCost: -5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := base
			c.Pricing = tt.p
			if (c.Validate() != nil) != tt.wantErr {
				t.Fatalf("Validate() err mismatch for Pricing %+v", tt.p)
			}
		})
	}
}

func TestPluginConfigTimeoutOrDefault(t *testing.T) {
	p := PluginConfig{Name: "mytool", Command: "x"}
	d, err := p.TimeoutOrDefault()
	if err != nil {
		t.Fatal(err)
	}
	if d != 60*time.Second {
		t.Fatalf("TimeoutOrDefault() = %s, want 60s default", d)
	}
	p.Timeout = "5s"
	d, err = p.TimeoutOrDefault()
	if err != nil {
		t.Fatal(err)
	}
	if d != 5*time.Second {
		t.Fatalf("TimeoutOrDefault() = %s, want 5s", d)
	}
}

func TestLoadMetricsFromYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	yaml := "metrics:\n  - name: weekend-usage\n    command: /usr/local/bin/assaio-metric-weekend\n    timeout: 30s\n"
	if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(c.Metrics))
	}
	m := c.Metrics[0]
	if m.Name != "weekend-usage" || m.Command != "/usr/local/bin/assaio-metric-weekend" {
		t.Fatalf("metric entry wrong: %+v", m)
	}
	timeout, err := m.TimeoutOrDefault()
	if err != nil || timeout != 30*time.Second {
		t.Fatalf("TimeoutOrDefault() = %v, %v; want 30s", timeout, err)
	}
}

func TestValidateRejectsInvalidMetricEntry(t *testing.T) {
	c := Config{Since: "30d", Format: "table", Metrics: []PluginConfig{{Name: "Bad_Name", Command: "/x"}}}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "metric") {
		t.Fatalf("Validate() = %v, want a metric-entry error", err)
	}
}

func TestValidateRejectsDuplicateMetricNames(t *testing.T) {
	c := Config{Since: "30d", Format: "table", Metrics: []PluginConfig{
		{Name: "a", Command: "/x"}, {Name: "a", Command: "/y"},
	}}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), `duplicate metric name "a"`) {
		t.Fatalf("Validate() = %v, want duplicate metric name error", err)
	}
}

func TestValidateRejectsDuplicatePluginNames(t *testing.T) {
	c := Config{Since: "30d", Format: "table", Plugins: []PluginConfig{
		{Name: "a", Command: "/x"}, {Name: "a", Command: "/y"},
	}}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), `duplicate plugin name "a"`) {
		t.Fatalf("Validate() = %v, want duplicate plugin name error", err)
	}
}

func TestValidateAllowsSameNameAcrossPluginsAndMetrics(t *testing.T) {
	c := Config{
		Since: "30d", Format: "table",
		Plugins: []PluginConfig{{Name: "mytool", Command: "/x"}},
		Metrics: []PluginConfig{{Name: "mytool", Command: "/x"}},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil: one binary may speak both protocols", err)
	}
}
