package forward

import (
	"strings"
	"testing"

	"github.com/coredns/caddy"
	"github.com/coredns/caddy/caddyfile"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin/dnstap"
)

func TestList(t *testing.T) {
	f := Forward{
		proxies: []*Proxy{{addr: "1.1.1.1:53"}, {addr: "2.2.2.2:53"}, {addr: "3.3.3.3:53"}},
		p:       &roundRobin{},
	}

	expect := []*Proxy{{addr: "2.2.2.2:53"}, {addr: "1.1.1.1:53"}, {addr: "3.3.3.3:53"}}
	got := f.List()

	if len(got) != len(expect) {
		t.Fatalf("Expected: %v results, got: %v", len(expect), len(got))
	}
	for i, p := range got {
		if p.addr != expect[i].addr {
			t.Fatalf("Expected proxy %v to be '%v', got: '%v'", i, expect[i].addr, p.addr)
		}
	}
}

func TestSetTapPlugin(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		disableDnstap bool
	}{
		{"Enable dnstap", `forward . 127.0.0.1
		dnstap /tmp/dnstap.sock full
		dnstap tcp://example.com:6000
		`, false},
		{"Disable dnstap", `forward . 127.0.0.1 {
			no_dnstap
		}
		dnstap /tmp/dnstap.sock full
		dnstap tcp://example.com:6000
		`, true},
	}
	for _, tc := range tests {
		stanzas := strings.Split(tc.input, "\n")
		var dnstapStanza string
		var fwdStanza string
		if tc.disableDnstap {
			dnstapStanza = strings.Join(stanzas[3:], "\n")
			fwdStanza = strings.Join(stanzas[0:3], "\n")
		} else {
			dnstapStanza = strings.Join(stanzas[1:], "\n")
			fwdStanza = stanzas[0]
		}
		c := caddy.NewTestController("dns", dnstapStanza)
		dnstapSetup, err := caddy.DirectiveAction("dns", "dnstap")
		if err != nil {
			t.Fatal(err)
		}
		if err = dnstapSetup(c); err != nil {
			t.Fatal(err)
		}
		c.Dispenser = caddyfile.NewDispenser("", strings.NewReader(fwdStanza))
		if err = setup(c); err != nil {
			t.Fatal(err)
		}
		dnsserver.NewServer("", []*dnsserver.Config{dnsserver.GetConfig(c)})
		f, ok := dnsserver.GetConfig(c).Handler("forward").(*Forward)
		if !ok {
			t.Fatal("Expected a forward plugin")
		}
		tap, ok := dnsserver.GetConfig(c).Handler("dnstap").(*dnstap.Dnstap)
		if !ok {
			t.Fatal("Expected a dnstap plugin")
		}
		f.SetTapPlugin(tap)
		if tc.disableDnstap {
			if len(f.tapPlugins) != 0 {
				t.Fatalf("Expected: 0 results, got: %v", len(f.tapPlugins))
			}
		} else {
			if len(f.tapPlugins) != 2 {
				t.Fatalf("Expected: 2 results, got: %v", len(f.tapPlugins))
			}
			if f.tapPlugins[0] != tap || tap.Next != f.tapPlugins[1] {
				t.Error("Unexpected order of dnstap plugins")
			}
		}
	}
}
