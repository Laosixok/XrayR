// Package specialprotocol adapts the existing SSPanel-Metron API to the
// pinned XrayRP AnyTLS and Hysteria2 runtimes. The old XrayR controller and
// panel endpoints remain independent of the specialized runtime API.
package specialprotocol

import (
	"encoding/json"
	"fmt"

	rpapi "github.com/Mtoly/XrayRP/api"
	rpanytls "github.com/Mtoly/XrayRP/service/anytls"
	rpcontroller "github.com/Mtoly/XrayRP/service/controller"
	rphy2 "github.com/Mtoly/XrayRP/service/hysteria2"

	"github.com/XrayR-project/XrayR/api"
	"github.com/XrayR-project/XrayR/service"
	"github.com/XrayR-project/XrayR/service/controller"
)

type panelBridge struct{ api.API }

func copyValue[T any](value any) (*T, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result T
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
func mappedError(err error) error {
	if err == nil {
		return nil
	}
	switch err.Error() {
	case api.NodeNotModified:
		return rpapi.ErrNodeNotModified
	case api.UserNotModified:
		return rpapi.ErrUserNotModified
	case api.RuleNotModified:
		return rpapi.ErrRuleNotModified
	}
	return err
}
func (b panelBridge) Describe() rpapi.ClientInfo {
	c := b.API.Describe()
	return rpapi.ClientInfo{APIHost: c.APIHost, NodeID: c.NodeID, Key: c.Key, NodeType: c.NodeType}
}
func (b panelBridge) GetNodeInfo() (*rpapi.NodeInfo, error) {
	v, err := b.API.GetNodeInfo()
	if err != nil {
		return nil, mappedError(err)
	}
	if v == nil {
		return nil, fmt.Errorf("nil node response")
	}
	return copyValue[rpapi.NodeInfo](v)
}
func (b panelBridge) GetUserList() (*[]rpapi.UserInfo, error) {
	v, err := b.API.GetUserList()
	if err != nil {
		return nil, mappedError(err)
	}
	// SSPanel-Metron subscriptions authenticate with UUID, not the account password.
	result := make([]rpapi.UserInfo, 0)
	if v != nil {
		for _, u := range *v {
			result = append(result, rpapi.UserInfo{UID: u.UID, Email: u.Email, UUID: u.UUID, SpeedLimit: u.SpeedLimit, DeviceLimit: u.DeviceLimit})
		}
	}
	return &result, nil
}
func (b panelBridge) GetNodeRule() (*[]rpapi.DetectRule, error) {
	v, err := b.API.GetNodeRule()
	if err != nil {
		return nil, mappedError(err)
	}
	result := make([]rpapi.DetectRule, 0)
	if v != nil {
		for _, r := range *v {
			result = append(result, rpapi.DetectRule{ID: r.ID, Pattern: r.Pattern})
		}
	}
	return &result, nil
}
func (b panelBridge) ReportNodeStatus(v *rpapi.NodeStatus) error {
	out, err := copyValue[api.NodeStatus](v)
	if err != nil {
		return err
	}
	return b.API.ReportNodeStatus(out)
}
func (b panelBridge) ReportNodeOnlineUsers(v *[]rpapi.OnlineUser) error {
	out, err := copyValue[[]api.OnlineUser](v)
	if err != nil {
		return err
	}
	return b.API.ReportNodeOnlineUsers(out)
}
func (b panelBridge) ReportUserTraffic(v *[]rpapi.UserTraffic) error {
	out, err := copyValue[[]api.UserTraffic](v)
	if err != nil {
		return err
	}
	return b.API.ReportUserTraffic(out)
}
func (b panelBridge) ReportIllegal(v *[]rpapi.DetectResult) error {
	out, err := copyValue[[]api.DetectResult](v)
	if err != nil {
		return err
	}
	return b.API.ReportIllegal(out)
}

var _ rpapi.PanelClient = panelBridge{}

func New(client api.API, config *controller.Config, panelType string) (service.Service, error) {
	if client == nil || config == nil {
		return nil, fmt.Errorf("panel client and controller configuration are required")
	}
	if config.WSAdditionalTLSPort != 0 {
		return nil, fmt.Errorf("WSAdditionalTLSPort only applies to VLESS WebSocket nodes")
	}
	cfg, err := copyValue[rpcontroller.Config](config)
	if err != nil {
		return nil, err
	}
	cfg.GlobalPanelType = panelType
	bridge := panelBridge{client}
	switch client.Describe().NodeType {
	case "AnyTLS":
		return rpanytls.New(bridge, cfg), nil
	case "Hysteria2":
		return rphy2.New(bridge, cfg), nil
	default:
		return nil, fmt.Errorf("unsupported specialized protocol: %s", client.Describe().NodeType)
	}
}
