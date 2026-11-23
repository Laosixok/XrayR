# XrayR 1.0.0 维护与来源声明

本仓库为个人维护的 XrayR 分支，面向 SSPanel-Metron 对接与自用。

- 基于原 XrayR 最后可确认的开发源码维护。
- AnyTLS、Hysteria2 运行时复用固定提交版本的 Mtoly/XrayRP，具体版本见 go.mod。
- 本分支不是原 XrayR、XrayRP、SSPanel-Metron 或协议依赖的官方发行。
- 本仓库自己的源码按 LICENSE 中的 MPL-2.0 提供；第三方依赖保留各自许可证，详情见 THIRD_PARTY_NOTICES.md。
- 配置示例仅使用 example.com 保留域名。WS/WSS 示例端口分别为 80、443。发布包不包含个人域名、API 密钥或证书私钥。

SSPanel-Metron 的现有接口保持兼容，旧 XrayR 0.9.0 后端无需随本版本一起升级。
软件按许可证约定原样提供，不承诺可用性或特定用途适用性。
