package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPreparePreservesRealityAndXHTTP(t *testing.T) {
	raw := `{"outbounds":[{"protocol":"vless","settings":{"vnext":[{"address":"example.invalid","port":443,"users":[{"id":"secret-id"}]}]},"streamSettings":{"network":"xhttp","security":"reality","realitySettings":{"publicKey":"secret-key","shortId":"aabb","futureOption":7},"xhttpSettings":{"mode":"auto","extra":{"future":true}}}}],"dns":{"servers":["1.1.1.1"]},"routing":{"domainStrategy":"AsIs"},"inbounds":[{"listen":"0.0.0.0","port":9999}],"log":{"access":"sensitive.log","loglevel":"debug"}}`
	prepared, e := PrepareConfig([]byte(raw))
	if e != nil {
		t.Fatal(e)
	}
	var before, after map[string]json.RawMessage
	_ = json.Unmarshal([]byte(raw), &before)
	_ = json.Unmarshal(prepared, &after)
	for _, key := range []string{"outbounds", "dns", "routing"} {
		var a, b any
		_ = json.Unmarshal(before[key], &a)
		_ = json.Unmarshal(after[key], &b)
		aa, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		if string(aa) != string(bb) {
			t.Fatalf("changed %s", key)
		}
	}
	if strings.Contains(string(prepared), "0.0.0.0") || strings.Contains(string(prepared), "sensitive.log") {
		t.Fatal("unsafe input retained")
	}
	var in []struct {
		Listen string
		Port   int
		Tag    string
	}
	_ = json.Unmarshal(after["inbounds"], &in)
	if len(in) != 2 || in[0].Listen != "127.0.0.1" || in[0].Port != 10808 || in[0].Tag != "socks-in" || in[1].Port != 10809 {
		t.Fatalf("unexpected inbounds: %+v", in)
	}
}
func TestRawAndTCPAliasesNotRewritten(t *testing.T) {
	for _, network := range []string{"raw", "tcp", "xhttp"} {
		raw := `{"outbounds":[{"protocol":"vless","streamSettings":{"network":"` + network + `"}}]}`
		result, e := PrepareConfig([]byte(raw))
		if e != nil || !strings.Contains(string(result), `"`+network+`"`) {
			t.Fatalf("alias %s rewritten: %s %v", network, result, e)
		}
	}
}
func TestRejectInvalidAndServiceFeatures(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{}`, `{"outbounds":[]}`, `{"outbounds":[null]}`, `{"outbounds":[{}]}`, `{"outbounds":[{"protocol":"freedom"}],"api":{}}`, `{"outbounds":[{"protocol":"freedom"}],"browserForwarder":{"listenAddr":"0.0.0.0"}}`, `{"outbounds":[{"protocol":"freedom"}],"transport":{}}`} {
		if _, e := PrepareConfig([]byte(raw)); e == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
func TestInputSizeLimit(t *testing.T) {
	if _, e := PrepareConfig(make([]byte, maxConfigBytes+1)); e == nil {
		t.Fatal("accepted oversized input")
	}
}
