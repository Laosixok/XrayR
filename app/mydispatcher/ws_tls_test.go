package mydispatcher

import (
	"context"
	"regexp"
	"testing"

	"github.com/XrayR-project/XrayR/api"
	"github.com/XrayR-project/XrayR/common/limiter"
	"github.com/XrayR-project/XrayR/common/rule"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/transport"
)

func TestWSSharedLimits(t *testing.T) {
	d := &DefaultDispatcher{Limiter: limiter.New(), RuleManager: rule.New()}
	users := []api.UserInfo{{UID: 1, DeviceLimit: 1, SpeedLimit: 10000}}
	if err := d.Limiter.AddInboundLimiter("node", 0, &users, nil); err != nil {
		t.Fatal(err)
	}
	d.RegisterInboundAlias("node_ws_tls", "node")
	prepare := func(tag, ip string) bool {
		ctx := session.ContextWithInbound(context.Background(), &session.Inbound{Tag: tag, Source: net.TCPDestination(net.ParseAddress(ip), 1234), User: &protocol.MemoryUser{Email: "node||1"}})
		return d.PrepareOutboundLink(ctx, &transport.Link{Writer: buf.Discard})
	}
	if !prepare("node", "192.0.2.1") || !prepare("node_ws_tls", "192.0.2.1") {
		t.Fatal("same user/IP must work on both entries")
	}
	if prepare("node_ws_tls", "192.0.2.2") {
		t.Fatal("TLS entry bypassed the primary entry's device limit")
	}
	online, err := d.Limiter.GetOnlineDevice("node")
	if err != nil || len(*online) != 1 {
		t.Fatalf("duplicate online devices: %+v, %v", online, err)
	}
	value, _ := d.Limiter.InboundInfo.Load("node")
	buckets := 0
	value.(*limiter.InboundInfo).BucketHub.Range(func(_, _ interface{}) bool { buckets++; return true })
	if buckets != 1 {
		t.Fatalf("expected one shared rate bucket, got %d", buckets)
	}
}

func TestWSSharedAudit(t *testing.T) {
	d := &DefaultDispatcher{Limiter: limiter.New(), RuleManager: rule.New()}
	users := []api.UserInfo{{UID: 1}}
	if err := d.Limiter.AddInboundLimiter("node", 0, &users, nil); err != nil {
		t.Fatal(err)
	}
	d.RegisterInboundAlias("node_ws_tls", "node")
	d.RuleManager.UpdateRule("node", []api.DetectRule{{ID: 7, Pattern: regexp.MustCompile("blocked.example")}})
	for _, tag := range []string{"node", "node_ws_tls"} {
		ctx := session.ContextWithInbound(context.Background(), &session.Inbound{Tag: tag, Source: net.TCPDestination(net.ParseAddress("192.0.2.1"), 1234), User: &protocol.MemoryUser{Email: "node||1"}})
		ctx = session.ContextWithOutbounds(ctx, []*session.Outbound{{Target: net.TCPDestination(net.ParseAddress("blocked.example"), 443)}})
		if d.PrepareOutboundLink(ctx, &transport.Link{}) {
			t.Fatalf("audit bypass on %s", tag)
		}
	}
	results, err := d.RuleManager.GetDetectResult("node")
	if err != nil || len(*results) != 1 || (*results)[0].UID != 1 || (*results)[0].RuleID != 7 {
		t.Fatalf("audit result not shared: %+v, %v", results, err)
	}
}
