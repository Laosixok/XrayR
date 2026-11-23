package specialprotocol_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
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
	"github.com/XrayR-project/XrayR/api/sspanel"
	"github.com/XrayR-project/XrayR/common/mylego"
	"github.com/XrayR-project/XrayR/service/controller"
	"github.com/XrayR-project/XrayR/service/specialprotocol"
	anyclient "github.com/anytls/sing-anytls"
	hyclient "github.com/apernet/hysteria/core/v2/client"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
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

func TestSSPanelMetronProtocolRuntime(t *testing.T) {
	for _, kind := range []string{"AnyTLS", "Hysteria2"} {
		t.Run(kind, func(t *testing.T) {
			cert, key, roots := testCertificate(t)
			port := freePort(t)
			target, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer target.Close()
			go func() {
				for {
					conn, err := target.Accept()
					if err != nil {
						return
					}
					go func() { defer conn.Close(); io.Copy(conn, conn) }()
				}
			}()
			const id = "11111111-2222-4333-8444-555555555555"
			var enabled atomic.Bool
			enabled.Store(true)
			var uploaded, downloaded atomic.Int64
			nodeSort := 18
			if kind == "Hysteria2" {
				nodeSort = 17
			}
			mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/mod_mu/nodes/9/info":
					fmt.Fprintf(w, `{"ret":1,"data":{"sort":%d,"server":"node.example;port=%d&obfs=plain&up_mbps=0&down_mbps=0"}}`, nodeSort, port)
				case "/mod_mu/users":
					if enabled.Load() {
						fmt.Fprintf(w, `{"ret":1,"data":[{"id":1,"uuid":%q,"passwd":"not-the-protocol-password","node_connector":2}]}`, id)
					} else {
						fmt.Fprint(w, `{"ret":1,"data":[]}`)
					}
				case "/mod_mu/users/traffic":
					var body struct {
						Data []struct {
							UID  int   `json:"user_id"`
							Up   int64 `json:"u"`
							Down int64 `json:"d"`
						} `json:"data"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					for _, row := range body.Data {
						if row.UID != 1 {
							t.Errorf("wrong UID %d", row.UID)
						}
						uploaded.Add(row.Up)
						downloaded.Add(row.Down)
					}
					fmt.Fprint(w, `{"ret":1,"data":"ok"}`)
				default:
					fmt.Fprint(w, `{"ret":1,"data":[]}`)
				}
			}))
			defer mock.Close()
			apiClient := sspanel.New(&api.Config{APIHost: mock.URL, NodeID: 9, NodeType: kind})
			cfg := &controller.Config{ListenIP: "127.0.0.1", SendIP: "0.0.0.0", UpdatePeriodic: 1, DisableGetRule: true, CertConfig: &mylego.CertConfig{CertMode: "file", CertDomain: "node.example", CertFile: cert, KeyFile: key}}
			svc, err := specialprotocol.New(apiClient, cfg, "SSpanel")
			if err != nil {
				t.Fatal(err)
			}
			if err = svc.Start(); err != nil {
				t.Fatal(err)
			}
			defer svc.Close()
			echo := func(password string) error {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var conn net.Conn
				if kind == "AnyTLS" {
					client, err := anyclient.NewClient(ctx, anyclient.ClientConfig{Password: password, Logger: logger.NOP(), DisableReuse: true, IdleSessionCheckInterval: time.Second, IdleSessionTimeout: time.Second, DialOut: func(ctx context.Context) (net.Conn, error) {
						d := tls.Dialer{Config: &tls.Config{RootCAs: roots, ServerName: "node.example", MinVersion: tls.VersionTLS12}}
						return d.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
					}})
					if err != nil {
						return err
					}
					defer client.Close()
					conn, err = client.CreateProxy(ctx, M.ParseSocksaddr(target.Addr().String()))
					if err != nil {
						return err
					}
				} else {
					addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
					client, _, err := hyclient.NewClient(&hyclient.Config{ServerAddr: addr, Auth: password, TLSConfig: hyclient.TLSConfig{ServerName: "node.example", RootCAs: roots}})
					if err != nil {
						return err
					}
					defer client.Close()
					conn, err = client.TCP(target.Addr().String())
					if err != nil {
						return err
					}
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(2 * time.Second))
				payload := []byte("metron-protocol-echo")
				if _, err = conn.Write(payload); err != nil {
					return err
				}
				response := make([]byte, len(payload))
				if _, err = io.ReadFull(conn, response); err != nil {
					return err
				}
				if !bytes.Equal(response, payload) {
					return fmt.Errorf("wrong echo: %q", response)
				}
				return nil
			}
			if err = echo(id); err != nil {
				t.Fatal("UUID connection failed:", err)
			}
			if err = echo("not-the-protocol-password"); err == nil {
				t.Fatal("account password must not authenticate a UUID subscription")
			}
			deadline := time.Now().Add(6 * time.Second)
			for (uploaded.Load() == 0 || downloaded.Load() == 0) && time.Now().Before(deadline) {
				time.Sleep(25 * time.Millisecond)
			}
			if uploaded.Load() == 0 || downloaded.Load() == 0 {
				t.Fatalf("missing panel accounting: up=%d down=%d", uploaded.Load(), downloaded.Load())
			}
			enabled.Store(false)
			deadline = time.Now().Add(8 * time.Second)
			for {
				err = echo(id)
				if err != nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("revoked UUID still connects")
				}
				time.Sleep(100 * time.Millisecond)
			}
			enabled.Store(true)
			deadline = time.Now().Add(8 * time.Second)
			for {
				err = echo(id)
				if err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("restored UUID failed:", err)
				}
				time.Sleep(100 * time.Millisecond)
			}
		})
	}
}
