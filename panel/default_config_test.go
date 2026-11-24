package panel_test

import (
	"github.com/XrayR-project/XrayR/panel"
	"github.com/spf13/viper"
	"os"
	"testing"
)

func TestDefaultInstallConfig(t *testing.T) {
	a, err := os.ReadFile("../release/config/config.yml.example")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../release/config/config_sspanel_metron_default.yml.example")
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("default templates differ")
	}
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile("../release/config/config.yml.example")
	if err = v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	var cfg panel.Config
	if err = v.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.NodesConfig) != 2 {
		t.Fatal("expected SS and VLESS nodes")
	}
	ss, vl := cfg.NodesConfig[0], cfg.NodesConfig[1]
	if ss.ApiConfig.NodeType != "Shadowsocks" || !ss.ApiConfig.DisableCustomConfig {
		t.Fatal("invalid SS defaults")
	}
	if vl.ApiConfig.NodeType != "Vless" || !vl.ApiConfig.EnableVless || !vl.ApiConfig.DisableCustomConfig || vl.ApiConfig.VlessFlow != "" {
		t.Fatal("invalid VLESS defaults")
	}
	if vl.ControllerConfig.WSAdditionalTLSPort != 8443 || vl.ControllerConfig.EnableFallback || vl.ControllerConfig.EnableREALITY {
		t.Fatal("invalid dual-entry defaults")
	}
	for _, node := range cfg.NodesConfig {
		if node.ControllerConfig.CertConfig == nil || node.ControllerConfig.CertConfig.CertMode != "dns" {
			t.Fatal("certificate settings not decoded")
		}
	}
}
