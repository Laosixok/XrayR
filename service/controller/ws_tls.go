package controller

import (
	"fmt"
	"github.com/XrayR-project/XrayR/api"
)

func (c *Controller) wsTLSTag(tag string) string { return tag + "_ws_tls" }

func (c *Controller) userInboundTags(tag string) []string {
	tags := []string{tag}
	if c.config.WSAdditionalTLSPort != 0 && tag == c.Tag {
		tags = append(tags, c.wsTLSTag(tag))
	}
	return tags
}

func (c *Controller) validateWSAdditionalTLS(node *api.NodeInfo) error {
	port := c.config.WSAdditionalTLSPort
	if port == 0 {
		return nil
	}
	if !(node.NodeType == "Vless" || (node.NodeType == "V2ray" && node.EnableVless)) ||
		(node.TransportProtocol != "ws" && node.TransportProtocol != "websocket") || node.EnableTLS ||
		node.EnableREALITY || c.config.EnableREALITY || node.VlessFlow != "" {
		return fmt.Errorf("WSAdditionalTLSPort requires a plain VLESS WebSocket node without REALITY or flow")
	}
	if uint32(port) == node.Port {
		return fmt.Errorf("WSAdditionalTLSPort must differ from the panel node port")
	}
	if c.config.CertConfig == nil || c.config.CertConfig.CertMode == "none" || c.config.CertConfig.CertMode == "" {
		return fmt.Errorf("WSAdditionalTLSPort requires CertConfig with file, dns, http or tls mode")
	}
	return nil
}

func (c *Controller) addWSAdditionalTLS(node *api.NodeInfo) error {
	if c.config.WSAdditionalTLSPort == 0 {
		return nil
	}
	tlsNode := *node
	tlsNode.Port = uint32(c.config.WSAdditionalTLSPort)
	tlsNode.EnableTLS = true
	tag := c.wsTLSTag(c.Tag)
	config, err := InboundBuilder(c.config, &tlsNode, tag)
	if err != nil {
		return err
	}
	// Both handlers use identical user emails, hence the same traffic counters.
	// Canonicalize the inbound tag before dispatch for shared rules and limits.
	c.dispatcher.RegisterInboundAlias(tag, c.Tag)
	if err := c.addInbound(config); err != nil {
		c.dispatcher.RemoveInboundAlias(tag)
		return err
	}
	return nil
}
