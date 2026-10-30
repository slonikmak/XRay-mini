package core

import (
	"encoding/json"
	"errors"
)

const maxConfigBytes = 2 << 20

// PrepareConfig preserves accepted subtrees as raw JSON, including future transport fields.
// It deliberately accepts only a small top-level surface, never arbitrary local services.
func PrepareConfig(input []byte) ([]byte, error) {
	if len(input) > maxConfigBytes {
		return nil, errors.New("configuration exceeds 2 MiB")
	}
	var cfg map[string]json.RawMessage
	if json.Unmarshal(input, &cfg) != nil || cfg == nil {
		return nil, errors.New("configuration must be a JSON object")
	}
	for key := range cfg {
		switch key {
		case "inbounds", "outbounds", "dns", "routing", "policy", "log":
		default:
			return nil, errors.New("unsupported top-level configuration feature; allowed: outbounds, dns, routing, policy, inbounds, log")
		}
	}
	var out []json.RawMessage
	if json.Unmarshal(cfg["outbounds"], &out) != nil || len(out) == 0 {
		return nil, errors.New("configuration requires a nonempty outbounds array")
	}
	for _, raw := range out {
		var o map[string]json.RawMessage
		if json.Unmarshal(raw, &o) != nil || o == nil {
			return nil, errors.New("each outbound must be an object")
		}
		var protocol string
		if json.Unmarshal(o["protocol"], &protocol) != nil || protocol == "" {
			return nil, errors.New("each outbound requires a protocol")
		}
	}
	cfg["inbounds"] = json.RawMessage(`[{"tag":"socks-in","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":false}},{"tag":"http-in","listen":"127.0.0.1","port":10809,"protocol":"http","settings":{}}]`)
	cfg["log"] = json.RawMessage(`{"loglevel":"none"}`)
	return json.MarshalIndent(cfg, "", "  ")
}
