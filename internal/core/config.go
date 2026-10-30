package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
)

const maxConfigBytes = 2 << 20
const selectedOutboundTag = "selected-vless"

// ImportPreview contains section names only, never configuration values or tags.
type ImportPreview struct {
	DroppedSections []string `json:"droppedSections"`
	Warning         string   `json:"warning"`
}

func PreviewImport(configJSON string) (ImportPreview, error) {
	_, preview, err := selectOutbound([]byte(configJSON))
	return preview, err
}

// PrepareConfig constructs a new configuration, rather than amending the user's
// routing. The only outbound (and thus also Xray's default) is the selected VLESS.
// Transport bytes are preserved; protocol validation is delegated to pinned Xray.
func PrepareConfig(input []byte) ([]byte, error) {
	outbound, _, err := selectOutbound(input)
	if err != nil {
		return nil, err
	}
	outbound["tag"] = json.RawMessage(`"selected-vless"`)
	outbound["protocol"] = json.RawMessage(`"vless"`)
	cfg := map[string]any{
		"outbounds": []any{outbound},
		"inbounds":  json.RawMessage(`[{"tag":"socks-in","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":false}},{"tag":"http-in","listen":"127.0.0.1","port":10809,"protocol":"http","settings":{}}]`),
		"routing":   json.RawMessage(`{"domainStrategy":"AsIs","rules":[{"type":"field","inboundTag":["socks-in","http-in"],"outboundTag":"selected-vless"}]}`),
		"log":       json.RawMessage(`{"loglevel":"warning"}`),
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func selectOutbound(input []byte) (map[string]json.RawMessage, ImportPreview, error) {
	preview := ImportPreview{DroppedSections: []string{}, Warning: "Only one selected VLESS outbound and its streamSettings will be used. Xray Mini replaces inbounds, routing and logging; other sections and outbounds are not imported."}
	fail := func(message string) (map[string]json.RawMessage, ImportPreview, error) {
		return nil, preview, errors.New(message)
	}
	if len(input) > maxConfigBytes {
		return fail("configuration exceeds 2 MiB")
	}
	if err := rejectDuplicateKeys(input); err != nil {
		return fail("configuration must be unambiguous JSON without duplicate keys")
	}
	var cfg map[string]json.RawMessage
	if json.Unmarshal(input, &cfg) != nil || cfg == nil {
		return fail("configuration must be a JSON object")
	}
	var out []json.RawMessage
	if json.Unmarshal(cfg["outbounds"], &out) != nil || len(out) == 0 {
		return fail("configuration requires an outbounds array with exactly one VLESS outbound")
	}
	var selected map[string]json.RawMessage
	count := 0
	for _, raw := range out {
		var o map[string]json.RawMessage
		if json.Unmarshal(raw, &o) != nil || o == nil {
			return fail("each outbound must be an object")
		}
		var protocol string
		if json.Unmarshal(o["protocol"], &protocol) != nil || protocol == "" {
			return fail("each outbound requires a protocol")
		}
		if strings.EqualFold(protocol, "vless") {
			count++
			selected = o
		}
	}
	if count != 1 {
		return fail("configuration must contain exactly one VLESS outbound")
	}
	// Xray's struct decoder is case-insensitive; reject aliases which our selection
	// could interpret differently, and any outbound-level chaining/bind extensions.
	for key := range selected {
		switch key {
		case "protocol", "tag", "settings", "streamSettings", "mux", "targetStrategy":
		default:
			return fail("selected outbound contains an unsupported field or dependency")
		}
	}
	var stream map[string]json.RawMessage
	if json.Unmarshal(selected["streamSettings"], &stream) != nil || stream == nil {
		return fail("VLESS outbound requires an object streamSettings")
	}
	for key := range stream {
		switch key {
		case "network", "security", "realitySettings", "tlsSettings", "rawSettings", "tcpSettings", "xhttpSettings", "splithttpSettings":
		default:
			return fail("streamSettings contains an unsupported transport field or dependency")
		}
	}
	var network string
	if raw, ok := stream["network"]; ok {
		if json.Unmarshal(raw, &network) != nil {
			return fail("transport network must be a string")
		}
		switch network {
		case "raw", "tcp", "xhttp", "splithttp":
		default:
			return fail("supported transports are RAW and XHTTP")
		}
	}
	var security string
	if raw, ok := stream["security"]; ok {
		if json.Unmarshal(raw, &security) != nil {
			return fail("transport security must be a string")
		}
		switch security {
		case "reality", "tls", "none", "":
		default:
			return fail("unsupported transport security")
		}
	}
	var tree any
	raw, _ := json.Marshal(selected)
	_ = json.Unmarshal(raw, &tree)
	if unsafeDependency(tree) {
		return fail("selected VLESS uses unsupported chained, local-resource, reverse or split-download dependencies")
	}
	for key := range cfg {
		if key != "outbounds" {
			// Do not echo arbitrary user-controlled keys into diagnostics/UI.
			switch key {
			case "inbounds", "routing", "dns", "policy", "log", "api", "stats", "reverse", "transport", "observatory", "burstObservatory", "browserForwarder", "metrics":
				preview.DroppedSections = append(preview.DroppedSections, key)
			default:
				preview.DroppedSections = append(preview.DroppedSections, "other top-level sections")
			}
		}
	}
	if len(out) > 1 {
		preview.DroppedSections = append(preview.DroppedSections, "other outbounds")
	}
	sort.Strings(preview.DroppedSections)
	return selected, preview, nil
}

// This intentionally supports self-contained client profiles only. In particular,
// XHTTP downloadSettings can introduce another stream/endpoint and are not imported.
// Names are matched case-insensitively, like Xray's JSON struct decoder.
func unsafeDependency(v any) bool {
	switch value := v.(type) {
	case map[string]any:
		for key, child := range value {
			k := strings.ToLower(key)
			switch k {
			case "proxysettings", "dialerproxy", "sockopt", "sendthrough", "certificates", "masterkeylog", "keylog", "keylogfile", "downloadsettings", "reverse", "finalmask", "browserforwarder", "domainsocket", "interface":
				return true
			}
			if strings.HasSuffix(k, "file") || strings.HasSuffix(k, "filepath") {
				return true
			}
			if k == "address" {
				if address, ok := child.(string); ok && (strings.ContainsAny(address, `/\\`) || strings.HasPrefix(address, "@")) {
					return true
				}
			}
			if unsafeDependency(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if unsafeDependency(child) {
				return true
			}
		}
	}
	return false
}

// A streaming structural pass avoids encoding/json's last-key-wins ambiguity.
// Depth is bounded to keep hostile imports from exhausting the goroutine stack.
func rejectDuplicateKeys(input []byte) error {
	d := json.NewDecoder(bytes.NewReader(input))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return errors.New("excessive nesting")
		}
		tok, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				token, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := token.(string)
				if !ok {
					return errors.New("invalid key")
				}
				key = strings.ToLower(key)
				if seen[key] {
					return errors.New("duplicate key")
				}
				seen[key] = true
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
