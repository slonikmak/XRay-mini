package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Spec assertions, not changes to the implementation. Run against unchanged core.
func TestReviewSpecV02RejectUnsupportedImport(t *testing.T) {
	cases := map[string]string{
		"no_vless":               `{"outbounds":[{"protocol":"freedom"}]}`,
		"ambiguous_two_vless":    `{"outbounds":[{"protocol":"vless","streamSettings":{"network":"raw","security":"reality"}},{"protocol":"vless","streamSettings":{"network":"xhttp","security":"tls"}}]}`,
		"missing_streamSettings": `{"outbounds":[{"protocol":"vless"}]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := PrepareConfig([]byte(raw)); err == nil {
				t.Fatalf("FR-02 violation: accepted %s", raw)
			}
		})
	}
}

func TestReviewSpecV02RemoveDirectRouting(t *testing.T) {
	raw := `{"outbounds":[{"tag":"direct","protocol":"freedom"},{"tag":"target","protocol":"vless","streamSettings":{"network":"xhttp","security":"tls"}}],"routing":{"rules":[{"type":"field","inboundTag":["socks-in","http-in"],"outboundTag":"direct"}]}}`
	prepared, err := PrepareConfig([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Outbounds []struct {
			Protocol string `json:"protocol"`
			Tag      string `json:"tag"`
		} `json:"outbounds"`
		Routing struct {
			Rules []struct {
				OutboundTag string `json:"outboundTag"`
			} `json:"rules"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(prepared, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Outbounds) != 1 || result.Outbounds[0].Protocol != "vless" {
		t.Errorf("FR-02/03: expected one VLESS outbound; got %+v", result.Outbounds)
	}
	for _, rule := range result.Routing.Rules {
		if rule.OutboundTag == "direct" {
			t.Error("FR-04: original direct routing retained for the local proxy")
		}
	}
}

func TestReviewSpecV02CreatesExplicitRoute(t *testing.T) {
	raw := `{"outbounds":[{"protocol":"vless","streamSettings":{"network":"raw","security":"reality"}}]}`
	prepared, err := PrepareConfig([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(prepared, &result); err != nil {
		t.Fatal(err)
	}
	if _, exists := result["routing"]; !exists {
		t.Fatal("FR-03: no explicit routing generated")
	}
}

func TestReviewSpecV02CleansOrphanRuntime(t *testing.T) {
	root := t.TempDir()
	orphan := filepath.Join(root, "runtime-old-crashed-session")
	if err := os.Mkdir(orphan, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(orphan, "config.json")
	if err := os.WriteFile(cfg, []byte(`{"secret":"TEST_ONLY"}`), 0600); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Fatal("ADR-009: stale plaintext runtime config remains after startup")
	}
}

func TestReviewSpecV02RejectsUnverifiedExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xray.exe")
	if err := os.WriteFile(path, []byte("not an Xray binary"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateBinary(path); err == nil {
		t.Fatal("ADR-012: arbitrary regular file accepted without Xray identity/version/checksum verification")
	}
}
