package sspanel

import (
	"net/url"
	"testing"

	"github.com/XrayR-project/XrayR/api"
)

func TestSSPanelMetronNewProtocols(t *testing.T) {
	for _, tc := range []struct {
		kind   string
		sort   int
		server string
		port   uint32
	}{
		{"anytls", 18, "node.example;port=8443|server_name=tls.example|padding_scheme=" + url.QueryEscape(`["stop=8","0=30-30"]`), 8443},
		{"hy2", 17, "node.example;port=443&up_mbps=80&down_mbps=160&obfs=salamander&obfs_password=test-secret&ignore_client_bandwidth=1", 443},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			c := New(&api.Config{NodeType: tc.kind, NodeID: 9})
			n, err := c.parseMetronProtocol(&NodeInfoResponse{Sort: tc.sort, RawServerString: tc.server, SpeedLimit: 8})
			if err != nil {
				t.Fatal(err)
			}
			if n.Port != tc.port || !n.EnableTLS || n.SpeedLimit != 1000000 || n.NodeID != 9 {
				t.Fatalf("unexpected node: %+v", n)
			}
			if tc.sort == 18 {
				if n.AnyTLSConfig == nil || len(n.AnyTLSConfig.PaddingScheme) != 2 || n.SNI != "tls.example" {
					t.Fatalf("AnyTLS settings lost: %+v", n)
				}
			}
			if tc.sort == 17 {
				h := n.Hysteria2Config
				if h == nil || h.UpMbps != 80 || h.DownMbps != 160 || h.Obfs != "salamander" || h.ObfsPassword != "test-secret" || !h.IgnoreClientBandwidth {
					t.Fatalf("Hysteria2 settings lost: %+v", h)
				}
			}
		})
	}
}

func TestSSPanelMetronNewProtocolValidation(t *testing.T) {
	for _, tc := range []struct {
		kind   string
		sort   int
		server string
	}{
		{"AnyTLS", 17, "node;port=443"}, {"AnyTLS", 18, "node;port=0"},
		{"AnyTLS", 18, "node;tls_type=reality"}, {"AnyTLS", 18, "node;padding_scheme={}"},
		{"Hysteria2", 17, "node;obfs=salamander"}, {"Hysteria2", 17, "node;up_mbps=-1"},
		{"Hysteria2", 17, "node;port=65536"},
	} {
		c := New(&api.Config{NodeType: tc.kind})
		if _, err := c.parseMetronProtocol(&NodeInfoResponse{Sort: tc.sort, RawServerString: tc.server}); err == nil {
			t.Fatalf("accepted invalid %s server %q", tc.kind, tc.server)
		}
	}
}
