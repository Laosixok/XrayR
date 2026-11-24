# XrayR

当前维护版本：**1.0.0**。支持 SSPanel-Metron 对接、VLESS WS/WSS 双入口、AnyTLS 和 Hysteria2。

[下载程序](https://github.com/Laosixok/XrayR/releases/tag/1.0.0) · [SSPanel-Metron 配置说明](docs/SSPanel-Metron.md) · [维护与来源声明](NOTICE.md)

[![](https://img.shields.io/badge/TgChat-@XrayR讨论-blue.svg)](https://t.me/XrayR_project)
[![](https://img.shields.io/badge/Channel-@XrayR通知-blue.svg)](https://t.me/XrayR_channel)
![](https://img.shields.io/github/stars/XrayR-project/XrayR)
![](https://img.shields.io/github/forks/XrayR-project/XrayR)
![](https://github.com/XrayR-project/XrayR/actions/workflows/release.yml/badge.svg)
![](https://github.com/XrayR-project/XrayR/actions/workflows/docker.yml/badge.svg)
[![Github All Releases](https://img.shields.io/github/downloads/XrayR-project/XrayR/total.svg)]()


[English](https://github.com/XrayR-project/XrayR/blob/master/README-en.md)|[Iranian](https://github.com/XrayR-project/XrayR/blob/master/README_Fa.md)|[Vietnamese](https://github.com/XrayR-project/XrayR/blob/master/README-vi.md)

A Xray backend framework that can easily support many panels.

一个基于Xray的后端框架，支持V2ay,Trojan,Shadowsocks协议，极易扩展，支持多面板对接。

如果您喜欢本项目，可以右上角点个star+watch，持续关注本项目的进展。

使用教程：[详细使用教程](https://xrayr-project.github.io/XrayR-doc/)


## 免责声明

本仓库是个人维护的 XrayR 分支，当前版本为 1.0.0，用于学习与自用。项目继承原 XrayR，并固定复用 XrayRP 的 AnyTLS、Hysteria2 运行时；不代表原项目或相关依赖的官方发行。软件按原有许可证及依赖许可证提供，不承诺可用性或适用性。来源及许可证说明见 [NOTICE.md](NOTICE.md) 与 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。

## 功能与协议

| 协议 | SSPanel-Metron 对接 | 说明 |
|---|---|---|
| VMess / VLESS | 支持 | VLESS 可同时提供 WS 80 与 WSS 443 入口 |
| Trojan | 支持 | 配置 TLS 证书 |
| Shadowsocks | 支持 | 兼容旧式承载账号与用户配置 |
| AnyTLS | 支持 | sort=18，UUID 认证，可下发 padding_scheme |
| Hysteria2 | 支持 | sort=17，使用 UDP，可配置 salamander 混淆 |

支持用户同步、流量上报、在线 IP 限制、用户限速、审计规则和证书配置。
同一 VLESS 节点的 WS/WSS 双入口共用用户及流量统计。
面板接口保持兼容，原 XrayR 0.9.0 后端无需一起升级。
其他面板保留原有适配能力；新协议目前以 SSPanel-Metron 对接为准。

## 安装

### 一键安装或升级（Linux amd64 / systemd）

先安装 `curl`、`unzip`、`ca-certificates` 和 `coreutils`，再运行：

```bash
curl -fsSL https://raw.githubusercontent.com/Laosixok/XrayR/master/install.sh -o install.sh
sudo bash install.sh --version 1.0.0
```

省略 `--version` 时安装本仓库最新 Release。首次安装默认使用 VLESS WS/WSS 模板；
AnyTLS / Hysteria2 可添加 `--template protocols`。脚本校验 SHA256，不覆盖现有配置、
证书或 systemd 单元；首次安装不会自动启动。编辑 `/etc/XrayR/config.yml`，
填写面板信息并放置证书后执行：

```bash
sudo systemctl enable --now XrayR.service
sudo journalctl -u XrayR.service -f
```

重复运行安装命令即可升级，原来运行的服务会重启；可用 `--no-restart` 暂缓重启。
脚本只安装 Release 中实际提供的架构，不会给其他架构错误下载 amd64 程序。

### Docker 安装

使用仓库中的 Dockerfile 和 Compose 在本地构建，无需依赖预发布镜像。
完整步骤、配置挂载、证书和 UDP 设置见 [Docker 安装教程](docs/docker.md)。

### 手动安装或编译

从 [Release](https://github.com/Laosixok/XrayR/releases) 下载程序与 SHA256SUMS，
校验后解压并编辑配置文件：

```bash
sha256sum -c SHA256SUMS
unzip XrayR-linux-amd64.zip
./XrayR --config config.yml
```

源码编译需要 Go 1.27.4 或更新版本：

```bash
git clone https://github.com/Laosixok/XrayR.git
cd XrayR
go build -trimpath -ldflags "-s -w" -o XrayR .
./XrayR --config /etc/XrayR/config.yml
```

## 配置文件及详细使用教程

[详细使用教程](https://xrayr-project.github.io/XrayR-doc/)

## Thanks

* [Project X](https://github.com/XTLS/)
* [V2Fly](https://github.com/v2fly)
* [VNet-V2ray](https://github.com/ProxyPanel/VNet-V2ray)
* [Air-Universe](https://github.com/crossfw/Air-Universe)

## Licence

[Mozilla Public License Version 2.0](https://github.com/XrayR-project/XrayR/blob/master/LICENSE)

## Telgram

[XrayR后端讨论](https://t.me/XrayR_project)

[XrayR通知](https://t.me/XrayR_channel)

## Stargazers over time

[![Stargazers over time](https://starchart.cc/XrayR-project/XrayR.svg)](https://starchart.cc/XrayR-project/XrayR)


