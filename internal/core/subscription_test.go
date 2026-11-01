package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const syntheticLink = "vless://11111111-1111-4111-8111-111111111111@example.invalid:443?type=xhttp&security=tls&sni=sni.example.invalid&path=%2Ftest%3Fa%3D1&mode=packet-up&extra=%7B%22xmux%22%3A%7B%22maxConcurrency%22%3A%2216-32%22%7D%7D#Synthetic%20profile"

func TestSubscriptionFormatsAndMapping(t *testing.T) {
	list := strings.Repeat(syntheticLink+"\r\n", 5)
	for _, body := range []string{list, base64.StdEncoding.EncodeToString([]byte(list)), base64.RawStdEncoding.EncodeToString([]byte(list)), base64.URLEncoding.EncodeToString([]byte(list)), base64.RawURLEncoding.EncodeToString([]byte(list))} {
		profiles, err := decodeSubscription([]byte(body))
		if err != nil || len(profiles) != 5 {
			t.Fatalf("five-profile synthetic list: %v, count %d", err, len(profiles))
		}
		if profiles[0].Name != "Synthetic profile" || profiles[0].Transport != "XHTTP/TLS" {
			t.Fatal("metadata mapping failed")
		}
		var cfg map[string]any
		if err := json.Unmarshal([]byte(profiles[0].ConfigJSON), &cfg); err != nil {
			t.Fatal(err)
		}
		stream := cfg["outbounds"].([]any)[0].(map[string]any)["streamSettings"].(map[string]any)
		xhttp := stream["xhttpSettings"].(map[string]any)
		if xhttp["path"] != "/test?a=1" || xhttp["mode"] != "packet-up" || xhttp["extra"].(map[string]any)["xmux"].(map[string]any)["maxConcurrency"] != "16-32" {
			t.Fatal("XHTTP values were changed")
		}
		if stream["tlsSettings"].(map[string]any)["serverName"] != "sni.example.invalid" {
			t.Fatal("SNI changed")
		}
		prepared, err := PrepareConfig([]byte(profiles[0].ConfigJSON))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(prepared), `"listen": "127.0.0.1"`) || strings.Contains(string(prepared), `"protocol": "freedom"`) {
			t.Fatal("fixed loopback routing lost")
		}
	}
	for _, network := range []string{"raw", "tcp", "xhttp"} {
		link := "vless://synthetic%2Did@[2001:db8::1]:8443?type=" + network + "&security=reality&pbk=synthetic-key&fp=chrome&sid=abcd&spx=%2Fspider&pqv=synthetic-pq#IPv6"
		p, err := parseVLESSLink(link)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range []string{`"address":"2001:db8::1"`, `"password":"synthetic-key"`, `"spiderX":"/spider"`, `"mldsa65Verify":"synthetic-pq"`, `"id":"synthetic-id"`} {
			if !strings.Contains(p.ConfigJSON, value) {
				t.Fatal("REALITY/IPv6 mapping changed")
			}
		}
	}
}

func TestSubscriptionQueryUTF8AndEscapedUnicode(t *testing.T) {
	u, _ := url.Parse(syntheticLink)
	for _, parameter := range []string{"type", "security", "encryption", "flow", "sni", "fp", "alpn", "pbk", "sid", "spx", "pqv", "path", "host", "mode", "extra", "ech", "pcs", "vcn"} {
		q := u.Query()
		q.Set(parameter, "\xff")
		u.RawQuery = q.Encode()
		if _, err := parseVLESSLink(u.String()); err == nil || err.Error() != "VLESS parameter must be valid UTF-8 without control characters" {
			t.Fatalf("invalid decoded UTF-8 in %s was not rejected at the boundary: %v", parameter, err)
		}
		// Restore valid parameters so each iteration exercises the named value.
		u, _ = url.Parse(syntheticLink)
	}
	q := u.Query()
	q.Set("path", "/путь/東京?ключ=✓")
	u.RawQuery = q.Encode()
	profile, err := parseVLESSLink(u.String())
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Outbounds []struct {
			StreamSettings struct {
				XHTTPSettings struct {
					Path string `json:"path"`
				} `json:"xhttpSettings"`
			} `json:"streamSettings"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(profile.ConfigJSON), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Outbounds[0].StreamSettings.XHTTPSettings.Path != "/путь/東京?ключ=✓" {
		t.Fatal("valid escaped Unicode changed during conversion")
	}
}

func TestSubscriptionNameMatchesPersistenceBounds(t *testing.T) {
	for _, test := range []struct {
		name, fragment, want string
		reject               bool
	}{
		{name: "100-byte multibyte boundary", fragment: strings.Repeat("я", 50), want: strings.Repeat("я", 50)},
		{name: "multibyte overflow", fragment: strings.Repeat("я", 51), reject: true},
		{name: "trimmed multibyte boundary", fragment: "  " + strings.Repeat("я", 50) + "  ", want: strings.Repeat("я", 50)},
		{name: "whitespace only", fragment: " \u00a0 ", reject: true},
		{name: "trimmed name", fragment: "  Имя профиля  ", want: "Имя профиля"},
		{name: "existing rune bound", fragment: strings.Repeat("a", 81), reject: true},
		{name: "existing control rejection", fragment: "name\n", reject: true},
		{name: "missing fragment default", fragment: "", want: "Imported VLESS"},
	} {
		t.Run(test.name, func(t *testing.T) {
			u, _ := url.Parse(syntheticLink)
			u.Fragment = test.fragment
			u.RawFragment = ""
			profile, err := parseVLESSLink(u.String())
			if test.reject {
				if err == nil {
					t.Fatal("unpersistable candidate name accepted")
				}
				return
			}
			if err != nil || profile.Name != test.want {
				t.Fatalf("default name differs from stored name: %v", err)
			}
		})
	}
}

func TestSubscriptionRejectsAmbiguityAndUnsupportedInput(t *testing.T) {
	u, _ := url.Parse(syntheticLink)
	for _, query := range []string{
		u.RawQuery + "&type=raw", u.RawQuery + "&unknown=secret", "type=ws&security=tls", "type=xhttp&security=none", "type=raw&security=tls", "type=raw&security=reality&pbk=x", "type=xhttp&security=tls&allowInsecure=1", "type=xhttp&security=tls&path=", "type=xhttp&security=tls&mode=unknown", "type=xhttp&security=tls&pbk=x", "type=raw&security=reality&pbk=x&fp=chrome&extra=%7B%7D", "type=xhttp&security=tls&encryption=unknown", "type=xhttp&security=tls&alpn=h2,,h3", "type=xhttp&security=tls&path=%0Asecret", "type=xhttp&security=tls&extra=" + url.QueryEscape(`{"xmux":{},"XMUX":{}}`), "type=xhttp&security=tls&extra=" + url.QueryEscape(`{"downloadSettings":{"address":"secret"}}`), "type=xhttp&security=tls&extra=" + url.QueryEscape(`{"xmux":{"unknown":1}}`), "type=xhttp&security=tls&extra=" + url.QueryEscape(`{"xmux":{"maxConcurrency":"32-16"}}`), "type=xhttp&security=tls&extra=" + url.QueryEscape(`[]`),
	} {
		u.RawQuery = query
		if _, err := parseVLESSLink(u.String()); err == nil {
			t.Fatal("accepted ambiguous or unsupported query")
		} else if strings.Contains(err.Error(), "synthetic-key") {
			t.Fatal("secret in error")
		}
	}
	for _, body := range []string{"", "%%%%", "dmxlc3M6Ly9iYWQ=", syntheticLink + "\nvmess://secret", strings.Repeat(syntheticLink+"\n", 101), strings.Repeat("A", maxSubscriptionBytes+1), "vless://user:secret@example.invalid:443?type=xhttp&security=tls", "vless://id@example.invalid:0?type=xhttp&security=tls"} {
		if _, err := decodeSubscription([]byte(body)); err == nil {
			t.Fatal("accepted invalid subscription")
		}
	}
	if _, err := LoadSubscription(context.Background(), syntheticLink+"\n"+syntheticLink); err == nil {
		t.Fatal("single-link input accepted a list")
	}
}

func TestSubscriptionHTTPSBoundsTLSRedirectAndCancellation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			io.WriteString(w, base64.StdEncoding.EncodeToString([]byte(syntheticLink)))
		case "/large":
			io.WriteString(w, strings.Repeat("A", maxSubscriptionBytes+1))
		case "/redirect":
			http.Redirect(w, r, "http://example.invalid/secret", http.StatusFound)
		case "/wait":
			<-r.Context().Done()
		default:
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, "private server error")
		}
	}))
	defer server.Close()
	transport := server.Client().Transport
	if profiles, err := loadSubscription(context.Background(), server.URL+"/ok?synthetic-secret", transport); err != nil || len(profiles) != 1 {
		t.Fatalf("trusted TLS load: %v", err)
	}
	for _, path := range []string{"/large", "/redirect", "/forbidden?synthetic-secret"} {
		if _, err := loadSubscription(context.Background(), server.URL+path, transport); err == nil {
			t.Fatal("invalid response accepted")
		} else if strings.Contains(err.Error(), "synthetic-secret") || strings.Contains(err.Error(), "private server error") {
			t.Fatal("raw network data in error")
		}
	}
	if _, err := LoadSubscription(context.Background(), server.URL+"/ok"); err == nil {
		t.Fatal("untrusted certificate accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := loadSubscription(ctx, server.URL+"/wait", transport); err == nil || !strings.Contains(err.Error(), "canceled or timed out") {
		t.Fatal("timeout category lost")
	}
	for _, source := range []string{"http://example.invalid/secret", "https://user:secret@example.invalid", "https://example.invalid/#secret", "file:///secret"} {
		if _, err := LoadSubscription(context.Background(), source); err == nil {
			t.Fatal("unsafe source accepted")
		}
	}
}
