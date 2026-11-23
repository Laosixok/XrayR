package controller_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XrayR-project/XrayR/api"
	"github.com/XrayR-project/XrayR/common/mylego"
	"github.com/XrayR-project/XrayR/panel"
	"github.com/XrayR-project/XrayR/service/controller"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xtls/xray-core/infra/conf"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

func testCertificate(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "node.example"}, DNSNames: []string{"node.example"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath, keyPath := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err = os.WriteFile(certPath, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: raw}), 0600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certPEM)
	return certPath, keyPath, roots
}

func wsEcho(port int, secure bool, roots *x509.CertPool, targetPort int, id string) error {
	scheme := "ws"
	if secure {
		scheme = "wss"
	}
	dialer := websocket.Dialer{HandshakeTimeout: time.Second, TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: "node.example", MinVersion: tls.VersionTLS12}}
	conn, _, err := dialer.Dial(fmt.Sprintf("%s://127.0.0.1:%d/api/v1", scheme, port), http.Header{"Host": []string{"node.example."}})
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second))
	// VLESS v0, UUID, no addons, TCP, IPv4 destination followed by payload.
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	header := append([]byte{0}, uid[:]...)
	header = append(header, 0, 1)
	header = binary.BigEndian.AppendUint16(header, uint16(targetPort))
	header = append(header, 1, 127, 0, 0, 1)
	payload := []byte("dual-entry-echo")
	if err = conn.WriteMessage(websocket.BinaryMessage, append(header, payload...)); err != nil {
		return err
	}
	var received []byte
	for len(received) < len(payload)+2 {
		_, part, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		received = append(received, part...)
	}
	if !bytes.Equal(received, append([]byte{0, 0}, payload...)) {
		return fmt.Errorf("unexpected response: %x", received)
	}
	return nil
}

func TestWSDualEntryLifecycle(t *testing.T) {
	cert, key, roots := testCertificate(t)
	plainPort, tlsPort := freePort(t), freePort(t)
	for tlsPort == plainPort {
		tlsPort = freePort(t)
	}
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() {
		for {
			c, e := target.Accept()
			if e != nil {
				return
			}
			go func() { defer c.Close(); io.Copy(c, c) }()
		}
	}()
	id := "11111111-2222-4333-8444-555555555555"
	var enabled atomic.Bool
	enabled.Store(true)
	var reported atomic.Int64
	var reportedUp, reportedDown atomic.Int64
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/mod_mu/nodes/15/info":
			fmt.Fprintf(w, `{"ret":1,"data":{"sort":15,"server":"node.example;%d;0;ws;;path=/api/v1|host=node.example."}}`, plainPort)
		case "/mod_mu/users":
			if enabled.Load() {
				fmt.Fprintf(w, `{"ret":1,"data":[{"id":1,"uuid":%q,"node_connector":2}]}`, id)
			} else {
				fmt.Fprint(w, `{"ret":1,"data":[]}`)
			}
		case "/mod_mu/users/traffic":
			var body struct {
				Data []struct {
					UID      int   `json:"user_id"`
					Upload   int64 `json:"u"`
					Download int64 `json:"d"`
				} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if r.URL.Query().Get("node_id") != "15" || len(body.Data) != 1 || body.Data[0].UID != 1 {
				t.Errorf("unexpected traffic report: %+v", body)
			}
			for _, entry := range body.Data {
				reportedUp.Add(entry.Upload)
				reportedDown.Add(entry.Download)
			}
			reported.Add(1)
			fmt.Fprint(w, `{"ret":1,"data":"ok"}`)
		default:
			fmt.Fprint(w, `{"ret":1,"data":[]}`)
		}
	}))
	defer mock.Close()
	p := panel.New(&panel.Config{LogConfig: &panel.LogConfig{Level: "none"}, NodesConfig: []*panel.NodesConfig{{PanelType: "SSpanel", ApiConfig: &api.Config{APIHost: mock.URL, NodeID: 15, NodeType: "Vless"}, ControllerConfig: &controller.Config{OutboundFinalRules: []*conf.FreedomFinalRuleConfig{{Action: "allow", IP: &conf.StringList{"127.0.0.1/32"}}}, ListenIP: "127.0.0.1", SendIP: "0.0.0.0", UpdatePeriodic: 1, DisableGetRule: true, WSAdditionalTLSPort: uint16(tlsPort), CertConfig: &mylego.CertConfig{CertMode: "file", CertFile: cert, KeyFile: key, CertDomain: "node.example"}}}}})
	p.Start()
	defer p.Close()
	targetPort := target.Addr().(*net.TCPAddr).Port
	for _, secure := range []bool{false, true} {
		port := plainPort
		if secure {
			port = tlsPort
		}
		if err := wsEcho(port, secure, roots, targetPort, id); err != nil {
			t.Fatalf("initial secure=%v: %v", secure, err)
		}
	}
	c := p.Service[0].(*controller.Controller)
	online, err := c.GetOnlineDevice(c.Tag)
	if err != nil || len(*online) != 1 || (*online)[0].UID != 1 {
		t.Fatalf("WS/WSS must share one online IP: %+v, %v", online, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for reported.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if reported.Load() == 0 {
		t.Fatal("no user traffic report after WS/WSS traffic")
	}
	if reportedUp.Load() != 2*int64(len("dual-entry-echo")) || reportedDown.Load() != 2*int64(len("dual-entry-echo")) {
		t.Fatalf("combined traffic must count both entries exactly once: up=%d down=%d", reportedUp.Load(), reportedDown.Load())
	}
	enabled.Store(false)
	// The single controller must revoke users from both handlers.
	for _, secure := range []bool{false, true} {
		port := plainPort
		if secure {
			port = tlsPort
		}
		deadline = time.Now().Add(5 * time.Second)
		for {
			err := wsEcho(port, secure, roots, targetPort, id)
			if err != nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("revoked UUID still works, secure=%v", secure)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	enabled.Store(true)
	for _, secure := range []bool{false, true} {
		port := plainPort
		if secure {
			port = tlsPort
		}
		deadline = time.Now().Add(5 * time.Second)
		for {
			err := wsEcho(port, secure, roots, targetPort, id)
			if err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("restored UUID failed, secure=%v: %v", secure, err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}
