package sspanel

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/XrayR-project/XrayR/api"
)

// SSPanel-Metron uses sort 17 for Hysteria2, sort 18 for AnyTLS.
// The existing mod_mu response is consumed without changing the panel API.
func (c *APIClient) parseMetronProtocol(response *NodeInfoResponse) (*api.NodeInfo, error) {
	expectedSort := 17
	if c.NodeType == "AnyTLS" {
		expectedSort = 18
	}
	if response.Sort != expectedSort {
		return nil, fmt.Errorf("%s requires SSPanel-Metron node sort=%d, got %d", c.NodeType, expectedSort, response.Sort)
	}
	parts := strings.SplitN(response.RawServerString, ";", 2)
	if parts[0] == "" {
		return nil, fmt.Errorf("empty SSPanel-Metron server")
	}
	options := url.Values{}
	if len(parts) == 2 {
		var err error
		options, err = url.ParseQuery(strings.ReplaceAll(parts[1], "|", "&"))
		if err != nil {
			return nil, fmt.Errorf("invalid SSPanel-Metron protocol options: %w", err)
		}
	}
	port := 443
	if raw := options.Get("port"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		port = value
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}
	limit := response.SpeedLimit
	if c.SpeedLimit > 0 {
		limit = c.SpeedLimit
	}
	node := &api.NodeInfo{NodeType: c.NodeType, NodeID: c.NodeID, Port: uint32(port), Host: parts[0], SNI: options.Get("server_name"), EnableTLS: true, SpeedLimit: uint64(limit * 1000000 / 8)}
	if node.SNI == "" {
		node.SNI = parts[0]
	}
	if c.NodeType == "AnyTLS" {
		if mode := options.Get("tls_type"); mode != "" && mode != "tls" {
			return nil, fmt.Errorf("SSPanel-Metron AnyTLS subscriptions require tls_type=tls")
		}
		node.TransportProtocol = "tcp"
		config := &api.AnyTLSConfig{}
		if raw := options.Get("padding_scheme"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &config.PaddingScheme); err != nil || config.PaddingScheme == nil {
				return nil, fmt.Errorf("padding_scheme must be a JSON string array")
			}
		}
		node.AnyTLSConfig = config
		return node, nil
	}
	config := &api.Hysteria2Config{UpMbps: 100, DownMbps: 100, Obfs: options.Get("obfs"), ObfsPassword: options.Get("obfs_password")}
	if config.Obfs == "" || config.Obfs == "none" {
		config.Obfs = "plain"
	}
	if config.Obfs != "plain" && config.Obfs != "salamander" {
		return nil, fmt.Errorf("unsupported Hysteria2 obfs: %s", config.Obfs)
	}
	if config.Obfs == "salamander" && config.ObfsPassword == "" {
		return nil, fmt.Errorf("salamander requires obfs_password")
	}
	for key, dest := range map[string]*int{"up_mbps": &config.UpMbps, "down_mbps": &config.DownMbps} {
		if raw := options.Get(key); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid %s", key)
			}
			*dest = n
		}
	}
	if raw := options.Get("ignore_client_bandwidth"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid ignore_client_bandwidth")
		}
		config.IgnoreClientBandwidth = v
	}
	node.TransportProtocol = "udp"
	node.Hysteria2Config = config
	return node, nil
}
