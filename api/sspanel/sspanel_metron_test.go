package sspanel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/XrayR-project/XrayR/api"
)

func TestSSPanelMetronLegacyVLESS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/mod_mu/nodes/15/info":
			w.Write([]byte(`{"ret":1,"data":{"sort":15,"server":"cdn-auto.example;80;0;ws;;path=/api/v1|host=cdn-hk05.example.","type":"ss-panel-v3-mod_Uim"}}`))
		case "/mod_mu/users":
			if r.URL.Query().Get("node_id") != "15" {
				t.Error("missing node_id")
			}
			w.Write([]byte(`{"ret":1,"data":[{"id":1,"passwd":"secret","uuid":"11111111-2222-4333-8444-555555555555","node_connector":3},{"id":2,"passwd":"fallback","uuid":"","node_connector":2},{"id":3,"is_multi_user":1}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := New(&api.Config{APIHost: server.URL, NodeID: 15, NodeType: "Vless"})
	node, err := client.GetNodeInfo()
	if err != nil {
		t.Fatal(err)
	}
	if !node.EnableVless || node.EnableTLS || node.Port != 80 || node.Path != "/api/v1" || node.Host != "cdn-hk05.example." || node.TransportProtocol != "ws" {
		t.Fatalf("unexpected node: %+v", node)
	}
	users, err := client.GetUserList()
	if err != nil {
		t.Fatal(err)
	}
	if len(*users) != 2 || (*users)[0].DeviceLimit != 3 || (*users)[0].UUID != "11111111-2222-4333-8444-555555555555" {
		t.Fatalf("unexpected users: %+v", users)
	}
	// SSPanel-Metron User::getUuid uses UUID v3 (DNS namespace, id|password).
	if (*users)[1].UUID != "ff668242-599b-3784-81a1-37ea3a066174" {
		t.Fatalf("fallback UUID mismatch: %s", (*users)[1].UUID)
	}
}

func TestSSPanelMetronLegacySSCarrier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ret":1,"data":[{"id":1,"port":1234,"method":"aes-128-gcm"},{"id":2,"port":2080,"method":"chacha20-ietf-poly1305","is_multi_user":1}]}`))
	}))
	defer server.Close()
	client := New(&api.Config{APIHost: server.URL, NodeID: 1, NodeType: "Shadowsocks"})
	node, err := client.ParseSSNodeResponse(&NodeInfoResponse{})
	if err != nil {
		t.Fatal(err)
	}
	if node.Port != 2080 || node.CypherMethod != "chacha20-ietf-poly1305" {
		t.Fatalf("wrong carrier: %+v", node)
	}
}

func TestSSPanelMetronMalformedServer(t *testing.T) {
	client := New(&api.Config{NodeType: "Vless"})
	if _, err := client.ParseV2rayNodeResponse(&NodeInfoResponse{RawServerString: "host"}); err == nil {
		t.Fatal("expected malformed server error")
	}
	var users []UserResponse
	if err := json.Unmarshal([]byte(`[{"id":1,"node_iplimit":4,"node_connector":2}]`), &users); err != nil {
		t.Fatal(err)
	}
	parsed, err := client.ParseUserListResponse(&users)
	if err != nil || (*parsed)[0].DeviceLimit != 4 {
		t.Fatal("modern device limit must take precedence")
	}
}

func TestSSPanelMetronModernUnlimited(t *testing.T) {
	var users []UserResponse
	if err := json.Unmarshal([]byte(`[{"id":1,"node_iplimit":0,"node_connector":2}]`), &users); err != nil {
		t.Fatal(err)
	}
	client := New(&api.Config{})
	parsed, err := client.ParseUserListResponse(&users)
	if err != nil || (*parsed)[0].DeviceLimit != 0 {
		t.Fatal("explicit modern unlimited must not fall back to legacy limit")
	}
}
