package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestPreparePreservesRequiredTransportCombinations(t *testing.T) {
	for _, transport := range []struct{ network, security string }{{"raw", "reality"}, {"xhttp", "reality"}, {"xhttp", "tls"}} {
		t.Run(transport.network+"-"+transport.security, func(t *testing.T) {
			stream := `{"network":"` + transport.network + `","security":"` + transport.security + `","realitySettings":{"publicKey":"secret-key","shortId":"aabb","futureOption":7},"tlsSettings":{"serverName":"example.invalid","alpn":["h2"],"fingerprint":"chrome"},"xhttpSettings":{"mode":"auto","path":"/secret","extra":{"future":true}}}`
			raw := `{"outbounds":[{"tag":"user-tag","protocol":"vless","settings":{"vnext":[{"address":"example.invalid","port":443,"users":[{"id":"secret-id"}]}]},"streamSettings":` + stream + `}]}`
			prepared, err := PrepareConfig([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			var cfg struct{ Outbounds []map[string]json.RawMessage }
			if err = json.Unmarshal(prepared, &cfg); err != nil {
				t.Fatal(err)
			}
			var before, after any
			_ = json.Unmarshal([]byte(stream), &before)
			_ = json.Unmarshal(cfg.Outbounds[0]["streamSettings"], &after)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("transport altered")
			}
			again, err := PrepareConfig(prepared)
			if err != nil || string(again) != string(prepared) {
				t.Fatal("runtime preparation not idempotent", err)
			}
		})
	}
}

func TestPrepareForcesOnlyVLESSRoute(t *testing.T) {
	raw := `{"outbounds":[{"protocol":"freedom","tag":"direct"},{"protocol":"vless","tag":"old","streamSettings":{"network":"raw"}}],"routing":{"rules":[{"inboundTag":["socks-in","http-in"],"outboundTag":"direct"}]},"dns":{"servers":["1.1.1.1"]},"inbounds":[{"listen":"0.0.0.0"}],"log":{"access":"sensitive.log"},"api":{"tag":"secret-tag"}}`
	prepared, err := PrepareConfig([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Outbounds []struct{ Protocol, Tag string }
		Inbounds  []struct {
			Listen, Tag string
			Port        int
		}
		Routing struct {
			Rules []struct {
				Type        string
				InboundTag  []string
				OutboundTag string
			}
		}
	}
	if err = json.Unmarshal(prepared, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Outbounds) != 1 || cfg.Outbounds[0].Protocol != "vless" || cfg.Outbounds[0].Tag != selectedOutboundTag {
		t.Fatalf("unsafe outbounds: %+v", cfg.Outbounds)
	}
	if len(cfg.Inbounds) != 2 || cfg.Inbounds[0].Listen != "127.0.0.1" || cfg.Inbounds[1].Listen != "127.0.0.1" || cfg.Inbounds[0].Port != 10808 || cfg.Inbounds[1].Port != 10809 {
		t.Fatalf("unsafe listeners: %+v", cfg.Inbounds)
	}
	if len(cfg.Routing.Rules) != 1 || cfg.Routing.Rules[0].OutboundTag != selectedOutboundTag || !reflect.DeepEqual(cfg.Routing.Rules[0].InboundTag, []string{"socks-in", "http-in"}) {
		t.Fatalf("unsafe routing: %+v", cfg.Routing)
	}
	for _, forbidden := range []string{"freedom", "direct", "0.0.0.0", "sensitive.log", "1.1.1.1", "secret-tag"} {
		if strings.Contains(string(prepared), forbidden) {
			t.Fatalf("retained %s", forbidden)
		}
	}
	preview, err := PreviewImport(raw)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"api", "dns", "inbounds", "log", "other outbounds", "routing"}
	if !reflect.DeepEqual(preview.DroppedSections, expected) || preview.Warning == "" {
		t.Fatalf("missing import warning %+v", preview)
	}
}

func TestRejectInvalidOrAmbiguousConfiguration(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{}`, `{"outbounds":[]}`, `{"outbounds":[null]}`, `{"outbounds":[{}]}`, `{"outbounds":[{"protocol":"freedom"}]}`, `{"outbounds":[{"protocol":"vless"}]}`, `{"outbounds":[{"protocol":"vless","streamSettings":null}]}`, `{"outbounds":[{"protocol":"vless","streamSettings":[]}]}`, `{"outbounds":[{"protocol":"vless","streamSettings":{}},{"protocol":"vless","streamSettings":{}}]}`, `{"outbounds":[{"protocol":"vless","protocol":"freedom","streamSettings":{}}]}`, `{"outbounds":[{"protocol":"vless","streamSettings":{"network":"raw","Network":"xhttp"}}]}`, `{"outbounds":[{"protocol":"vless","streamSettings":{}}]} {}`} {
		if _, err := PrepareConfig([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestRejectUnsafeSelectedDependencies(t *testing.T) {
	for _, field := range []string{
		`"proxySettings":{"tag":"direct"}`, `"sendThrough":"127.0.0.1"`, `"unknownOutboundExtension":{}`,
		`"streamSettings":{"sockopt":{"dialerProxy":"direct"}}`,
		`"streamSettings":{"tlsSettings":{"certificates":[{"certificateFile":"C:/secret"}]}}`,
		`"streamSettings":{"tlsSettings":{"masterKeyLog":"C:/secret"}}`,
		`"streamSettings":{"xhttpSettings":{"extra":{"downloadSettings":{"address":"example.invalid"}}}}`,
		`"settings":{"vnext":[{"address":"@local-socket"}]}`,
		`"settings":{"vnext":[{"address":"C:\\socket"}]}`,
		`"settings":{"vnext":[{"users":[{"reverse":{"tag":"other"}}]}]}`,
	} {
		fields := `"protocol":"vless"`
		if !strings.HasPrefix(field, `"streamSettings"`) {
			fields += `,"streamSettings":{}`
		}
		raw := `{"outbounds":[{` + fields + `,` + field + `}]}`
		if _, err := PrepareConfig([]byte(raw)); err == nil {
			t.Errorf("accepted unsafe dependency: %s", raw)
		}
	}
}

func TestRawAndTCPAliasesNotRewritten(t *testing.T) {
	for _, network := range []string{"raw", "tcp", "xhttp"} {
		result, err := PrepareConfig([]byte(`{"outbounds":[{"protocol":"vless","streamSettings":{"network":"` + network + `"}}]}`))
		if err != nil || !strings.Contains(string(result), `"`+network+`"`) {
			t.Fatalf("alias %s rewritten: %s %v", network, result, err)
		}
	}
}
func TestImportPreviewDoesNotEchoSecrets(t *testing.T) {
	p, err := PreviewImport(`{"secret-user-controlled-key":"secret-value","outbounds":[{"protocol":"vless","streamSettings":{}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), "secret-") {
		t.Fatal("preview leaked input")
	}
}
func TestInputSizeAndDepthLimit(t *testing.T) {
	if _, err := PrepareConfig(make([]byte, maxConfigBytes+1)); err == nil {
		t.Fatal("accepted oversized input")
	}
	if err := rejectDuplicateKeys([]byte(strings.Repeat("[", 100) + "0" + strings.Repeat("]", 100))); err == nil {
		t.Fatal("accepted excessive nesting")
	}
}
