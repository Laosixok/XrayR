# SSPanel-Metron 兼容与 VLESS 双入口

本次仅升级 XrayR，SSPanel-Metron 面板无需修改，原 XrayR 0.9.0 可继续使用原有接口。

兼容 SSPanel-Metron `/mod_mu` 的旧式节点 `server` 配置、`node_connector` 设备限制和
`is_multi_user` Shadowsocks 承载账号；用户 UUID 为空时，按面板 `getUuid()`
相同规则生成 UUID v3（DNS namespace，`用户ID|passwd`）。
现代接口的 `node_iplimit` 优先，包括明确的 0（不限设备数）。

## 同一个节点提供 WS 和 WSS

使用 `release/config/config_sspanel_metron_ws_tls.yml.example`，替换面板地址、密钥、节点 ID 和证书路径。
面板中已有的 VLESS 节点保持不变，示例 server：

```text
cdn.example.com;80;0;ws;;path=/api/v1|host=node.example.com.
```

后端监听面板指定的 80（VLESS + WS），同时监听配置中的
`WSAdditionalTLSPort: 443`（VLESS + WS + TLS）。两个入口使用相同 UUID、
WS 路径和 Host，共享用户更新、流量计数、限速、在线 IP 限制及审计规则。
流量由同一个控制器向同一节点上报，面板只计算一次倍率。

此选项默认关闭，只接受无 TLS、无 REALITY、无 flow 的 VLESS WebSocket 主节点。
证书使用已有 CertConfig，支持 file、dns、http、tls 模式；自动证书续期也包含新增入口。
端口冲突、未配置证书等会在启动时报错。

客户端保留原节点内容，将端口改为 443，并添加：

```yaml
port: 443
tls: true
servername: node.example.com
network: ws
ws-opts:
  path: /api/v1
  headers:
    Host: node.example.com.
```

UUID、server、WS path 和 Host 保持原值。TLS SNI 使用证书覆盖的域名。
WS 入口未启用传输层 TLS；WSS 入口启用了 TLS。

## CDN 回源

客户端 TLS 连接 CDN 时，证书校验对象是 CDN 边缘站点。若 CDN 在边缘终止 TLS，
再用 HTTP 回源 80，源站无需使用新增的 443 入口。
若 CDN 以 HTTPS 回源 443，需配置正确的回源 SNI、证书和端口。
仅修改客户端并不能自动改变 CDN 回源设置；以上两种方式都应按实际 CDN 配置验证。
本次不自动修改面板订阅，现有 80 订阅及旧版后端保持原样。

## AnyTLS 与 Hysteria2

现有 SSPanel-Metron 节点类型保持不变：AnyTLS 为 sort=18，Hysteria2 为 sort=17。
两者仍使用 `/mod_mu/nodes/{id}/info`、`/mod_mu/users` 和原流量、在线用户上报接口，
不要求修改面板，也不会改变旧 XrayR 0.9.0 的接口。

- AnyTLS：`NodeType: AnyTLS`，读取 `port`、`server_name`、`padding_scheme`。
  `padding_scheme` 为 JSON 字符串数组；当前面板订阅仅适用 TLS，不启用 REALITY。
- Hysteria2：`NodeType: Hysteria2`（兼容 `hy2`），读取 `port`、`up_mbps`、
  `down_mbps`、`obfs`、`obfs_password`、`ignore_client_bandwidth`。
  默认无混淆；使用 salamander 时必须提供混淆密码。开放对应 UDP 端口。
- 认证密码使用面板订阅中的 UUID，不接受用户登录密码作为额外协议密码。
- 两者配置独立的 `CertConfig`；`WSAdditionalTLSPort` 只用于 VLESS WebSocket 节点。

示例：`release/config/config_sspanel_metron_protocols.yml.example`。

## 上游代码来源

新增协议复用 [Mtoly/XrayRP](https://github.com/Mtoly/XrayRP) 的 AnyTLS 和 Hysteria2
运行时，依赖固定在提交 `4a54ac6e91242952dbc7156c0f6e748301952c42`。
适配边界见 `service/specialprotocol`。AnyTLS 运行在内嵌 sing-box 中，
Hysteria2 使用 Hysteria core/extras；不是只给 Xray 配置添加协议名称。
这次不接入 XrayRP 的其他面板模式、TUIC、机器管理及管理接口。
后续升级先更新固定依赖，再运行 SSPanel-Metron 接口、新协议实际连接和 VLESS 双入口回归测试。
编译需要 Go 1.27.4 或更新版本。

### 新核心的私网访问策略

升级后的 Xray-core 默认阻止部分代理协议访问私网目的地址。普通公网节点不需要调整。
确需访问自建内网服务时，可在对应节点的 `ControllerConfig.OutboundFinalRules` 中
显式配置允许范围；不要用全放行规则替代明确的目的地址范围。
回归测试只放行 `127.0.0.1/32`，用于连接本机测试服务。
