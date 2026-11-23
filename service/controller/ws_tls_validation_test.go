package controller

import (
	"github.com/XrayR-project/XrayR/api"
	"github.com/XrayR-project/XrayR/common/mylego"
	"testing"
)

func TestWSAdditionalTLSValidation(t *testing.T) {
	c := &Controller{config: &Config{}}
	if err := c.validateWSAdditionalTLS(&api.NodeInfo{NodeType: "Shadowsocks"}); err != nil {
		t.Fatal("disabled option must not affect legacy nodes:", err)
	}
	c.config.WSAdditionalTLSPort = 443
	c.config.CertConfig = &mylego.CertConfig{CertMode: "file"}
	node := api.NodeInfo{NodeType: "Vless", Port: 80, TransportProtocol: "ws"}
	if err := c.validateWSAdditionalTLS(&node); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*api.NodeInfo){
		func(n *api.NodeInfo) { n.Port = 443 }, func(n *api.NodeInfo) { n.EnableTLS = true },
		func(n *api.NodeInfo) { n.TransportProtocol = "tcp" }, func(n *api.NodeInfo) { n.VlessFlow = "xtls-rprx-vision" },
		func(n *api.NodeInfo) { n.EnableREALITY = true }, func(n *api.NodeInfo) { n.NodeType = "Vmess" },
	} {
		bad := node
		change(&bad)
		if err := c.validateWSAdditionalTLS(&bad); err == nil {
			t.Fatalf("accepted invalid node: %+v", bad)
		}
	}
	c.config.CertConfig.CertMode = "none"
	if err := c.validateWSAdditionalTLS(&node); err == nil {
		t.Fatal("accepted missing TLS certificate mode")
	}
}
