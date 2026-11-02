package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxSubscriptionBytes = 2 << 20
const maxSubscriptionProfiles = 100
const maxShareLinkBytes = 64 << 10

// SubscriptionProfile stays inside Go. The facade returns separate metadata DTOs.
type SubscriptionProfile struct {
	Name       string
	Transport  string
	ConfigJSON string `json:"-"`
}

// LoadSubscription accepts only an explicit HTTPS source or a single VLESS URI.
// No URL, response body, or transport error is ever included in returned errors.
func LoadSubscription(ctx context.Context, source string) ([]SubscriptionProfile, error) {
	return loadSubscription(ctx, source, nil)
}

func loadSubscription(ctx context.Context, source string, transport http.RoundTripper) ([]SubscriptionProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	source = strings.TrimSpace(source)
	if len(source) > maxShareLinkBytes {
		return nil, errors.New("Subscription source is too large")
	}
	if ctx.Err() != nil {
		return nil, errors.New("Subscription loading canceled or timed out")
	}
	if strings.HasPrefix(source, "vless://") {
		profile, err := parseVLESSLink(source)
		if err != nil {
			return nil, err
		}
		return []SubscriptionProfile{profile}, nil
	}
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("Enter an HTTPS subscription URL or one supported VLESS link")
	}
	if transport == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.Proxy = nil // Explicit source only; no ambient proxy credentials or routing.
		defer t.CloseIdleConnections()
		transport = t
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect rejected") }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, errors.New("Invalid HTTPS subscription URL")
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("Subscription loading canceled or timed out")
		}
		return nil, errors.New("Subscription HTTPS request failed; check TLS, network and redirects")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Subscription server did not return success")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSubscriptionBytes+1))
	if ctx.Err() != nil {
		return nil, errors.New("Subscription loading canceled or timed out")
	}
	if err != nil {
		return nil, errors.New("Subscription response could not be read")
	}
	if len(body) > maxSubscriptionBytes {
		return nil, errors.New("Subscription exceeds 2 MiB")
	}
	return decodeSubscription(body)
}

func decodeSubscription(body []byte) ([]SubscriptionProfile, error) {
	if len(body) > maxSubscriptionBytes {
		return nil, errors.New("Subscription exceeds 2 MiB")
	}
	content := strings.TrimSpace(string(body))
	if !strings.HasPrefix(content, "vless://") {
		encoded := strings.Join(strings.Fields(content), "")
		var decoded []byte
		var err error
		for _, encoding := range []*base64.Encoding{base64.StdEncoding.Strict(), base64.RawStdEncoding.Strict(), base64.URLEncoding.Strict(), base64.RawURLEncoding.Strict()} {
			decoded, err = encoding.DecodeString(encoded)
			if err == nil {
				break
			}
		}
		if err != nil || len(decoded) == 0 {
			return nil, errors.New("Subscription must contain a plain or Base64 VLESS list")
		}
		content = strings.TrimSpace(string(decoded))
	}
	if len(content) == 0 || !utf8.ValidString(content) {
		return nil, errors.New("Subscription is empty or malformed")
	}
	lines := strings.Split(content, "\n")
	profiles := make([]SubscriptionProfile, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(profiles) >= maxSubscriptionProfiles {
			return nil, errors.New("Subscription exceeds 100 profiles")
		}
		profile, err := parseVLESSLink(line)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 {
		return nil, errors.New("Subscription is empty or malformed")
	}
	return profiles, nil
}

func parseVLESSLink(link string) (SubscriptionProfile, error) {
	fail := func(message string) (SubscriptionProfile, error) { return SubscriptionProfile{}, errors.New(message) }
	if len(link) > maxShareLinkBytes || strings.IndexFunc(link, unicode.IsSpace) >= 0 {
		return fail("VLESS link is malformed or too large")
	}
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "vless" || u.User == nil || u.User.Username() == "" || u.Path != "" || u.Opaque != "" {
		return fail("VLESS link requires user ID, host and port")
	}
	if _, password := u.User.Password(); password {
		return fail("VLESS link user information is unsupported")
	}
	if !utf8.ValidString(u.User.Username()) || strings.IndexFunc(u.User.Username(), unicode.IsControl) >= 0 {
		return fail("VLESS user ID is malformed")
	}
	host := u.Hostname()
	if host == "" || strings.ContainsAny(host, "/\\@%") || strings.IndexFunc(host, func(r rune) bool { return r > 127 || unicode.IsControl(r) }) >= 0 {
		return fail("VLESS host must be an IP address or ASCII domain")
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return fail("VLESS IPv6 address is invalid")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return fail("VLESS port must be between 1 and 65535")
	}
	if strings.Contains(u.RawQuery, "+") {
		return fail("VLESS parameter values must escape plus signs")
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return fail("VLESS parameters are malformed")
	}
	for key, values := range q {
		if len(values) != 1 {
			return fail("Duplicate VLESS parameters are unsupported")
		}
		switch key {
		case "type", "security", "encryption", "flow", "sni", "fp", "alpn", "pbk", "sid", "spx", "pqv", "path", "host", "mode", "extra", "ech", "pcs", "vcn":
		default:
			return fail("VLESS link contains an unsupported parameter")
		}
		if !utf8.ValidString(values[0]) || strings.IndexFunc(values[0], unicode.IsControl) >= 0 {
			return fail("VLESS parameter must be valid UTF-8 without control characters")
		}
	}
	network := q.Get("type")
	if network == "" {
		if q.Has("type") {
			return fail("VLESS transport cannot be empty")
		}
		network = "raw"
	}
	if network == "tcp" {
		network = "raw"
	}
	if network != "raw" && network != "xhttp" {
		return fail("Supported VLESS transports are RAW and XHTTP")
	}
	security := q.Get("security")
	if security != "reality" && !(network == "xhttp" && security == "tls") {
		return fail("Supported VLESS combinations are RAW/REALITY and XHTTP/REALITY or TLS")
	}
	encryption := "none"
	if q.Has("encryption") {
		encryption = q.Get("encryption")
		if encryption != "none" {
			return fail("Supported VLESS link encryption is none")
		}
	}
	user := map[string]any{"id": u.User.Username(), "encryption": encryption}
	if q.Has("flow") {
		flow := q.Get("flow")
		if flow != "" && (flow != "xtls-rprx-vision" || network != "raw") {
			return fail("VLESS flow is unsupported for this transport")
		}
		user["flow"] = flow
	}
	tls := map[string]any{"fingerprint": "chrome"}
	for param, field := range map[string]string{"sni": "serverName", "fp": "fingerprint"} {
		if q.Has(param) {
			if q.Get(param) == "" {
				return fail("TLS server name and fingerprint cannot be empty")
			}
			tls[field] = q.Get(param)
		}
	}
	if q.Has("alpn") {
		alpn := strings.Split(q.Get("alpn"), ",")
		for _, value := range alpn {
			if value == "" || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
				return fail("TLS ALPN list is malformed")
			}
		}
		tls["alpn"] = alpn
	}
	if security == "reality" {
		if q.Get("pbk") == "" || q.Get("fp") == "" {
			return fail("REALITY requires password and fingerprint")
		}
		for param, field := range map[string]string{"pbk": "password", "sid": "shortId", "spx": "spiderX", "pqv": "mldsa65Verify"} {
			if q.Has(param) {
				tls[field] = q.Get(param)
			}
		}
		for _, param := range []string{"ech", "pcs", "vcn", "alpn"} {
			if q.Has(param) {
				return fail("TLS-only parameter is unsupported with REALITY")
			}
		}
	} else {
		for _, param := range []string{"pbk", "sid", "spx", "pqv"} {
			if q.Has(param) {
				return fail("REALITY-only parameter is unsupported with TLS")
			}
		}
		for param, field := range map[string]string{"ech": "echConfigList", "pcs": "pinnedPeerCertSha256", "vcn": "verifyPeerCertByName"} {
			if q.Has(param) {
				tls[field] = q.Get(param)
			}
		}
	}
	stream := map[string]any{"network": network, "security": security, security + "Settings": tls}
	if network == "xhttp" {
		xhttp := map[string]any{}
		for _, param := range []string{"path", "host", "mode"} {
			if q.Has(param) {
				if q.Get(param) == "" {
					return fail("XHTTP path, host and mode cannot be empty")
				}
				xhttp[param] = q.Get(param)
			}
		}
		if q.Has("mode") {
			switch q.Get("mode") {
			case "auto", "packet-up", "stream-up", "stream-one":
			default:
				return fail("Unsupported XHTTP mode")
			}
		}
		if q.Has("extra") {
			raw := []byte(q.Get("extra"))
			var extra map[string]json.RawMessage
			if rejectDuplicateKeys(raw) != nil || json.Unmarshal(raw, &extra) != nil || extra == nil {
				return fail("XHTTP extra must be an unambiguous JSON object")
			}
			// Only the self-contained XMUX subset is accepted. Unknown fields
			// cannot be silently ignored by Xray's permissive JSON decoder.
			for key, value := range extra {
				if key != "xmux" {
					return fail("Unsupported XHTTP extra field; only XMUX is accepted")
				}
				var xmux map[string]json.RawMessage
				if json.Unmarshal(value, &xmux) != nil || xmux == nil {
					return fail("XHTTP XMUX must be an object")
				}
				for field, rawValue := range xmux {
					switch field {
					case "maxConcurrency", "maxConnections", "cMaxReuseTimes", "hMaxRequestTimes", "hMaxReusableSecs", "hKeepAlivePeriod":
					default:
						return fail("Unsupported XHTTP XMUX field")
					}
					var number int64
					if json.Unmarshal(rawValue, &number) == nil {
						if number < 0 && field != "hKeepAlivePeriod" {
							return fail("XHTTP XMUX range is invalid")
						}
						continue
					}
					var value string
					if field == "hKeepAlivePeriod" || json.Unmarshal(rawValue, &value) != nil {
						return fail("XHTTP XMUX value must be an integer or range")
					}
					bounds := strings.Split(value, "-")
					if len(bounds) != 2 {
						return fail("XHTTP XMUX range is invalid")
					}
					lower, err1 := strconv.ParseUint(bounds[0], 10, 32)
					upper, err2 := strconv.ParseUint(bounds[1], 10, 32)
					if err1 != nil || err2 != nil || lower > upper {
						return fail("XHTTP XMUX range is invalid")
					}
				}
			}
			xhttp["extra"] = extra
		}
		stream["xhttpSettings"] = xhttp
	} else {
		for _, param := range []string{"path", "host", "mode", "extra"} {
			if q.Has(param) {
				return fail("XHTTP parameter is unsupported with RAW")
			}
		}
	}
	cfg := map[string]any{"outbounds": []any{map[string]any{"protocol": "vless", "settings": map[string]any{"vnext": []any{map[string]any{"address": host, "port": port, "users": []any{user}}}}, "streamSettings": stream}}}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fail("VLESS link could not be converted")
	}
	if _, err := PrepareConfig(raw); err != nil {
		return fail("VLESS link uses unsupported configuration dependencies")
	}
	name := u.Fragment
	if name == "" {
		name = "Imported VLESS"
	}
	if !utf8.ValidString(name) || len([]rune(name)) > 80 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fail("VLESS profile name must be at most 80 characters without controls")
	}
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 100 {
		return fail("VLESS profile name must be 1–100 bytes after trimming")
	}
	return SubscriptionProfile{Name: name, Transport: strings.ToUpper(network) + "/" + strings.ToUpper(security), ConfigJSON: string(raw)}, nil
}
