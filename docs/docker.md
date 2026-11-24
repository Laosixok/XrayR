# Docker 安装

本教程使用 Linux 主机的 Docker Engine 与 Compose 插件，从本仓库构建镜像。
主机网络模式直接监听面板指定端口；它不使用 `ports` 映射。
参考 [Docker host 网络说明](https://docs.docker.com/engine/network/drivers/host/)。

## 准备源码与配置

```bash
git clone https://github.com/Laosixok/XrayR.git
cd XrayR
sudo install -d -m 0700 /opt/xrayr/config
sudo cp release/config/config_sspanel_metron_default.yml.example /opt/xrayr/config/config.yml
sudo cp release/config/{dns.json,route.json,custom_inbound.json,custom_outbound.json,rulelist,geoip.dat,geosite.dat} /opt/xrayr/config/
sudo chmod 0600 /opt/xrayr/config/*
```

AnyTLS / Hysteria2 改用 `config_sspanel_metron_protocols.yml.example` 作为 config.yml。
模板只是示例，启动前填写自己的面板地址、密钥和实际节点 ID。
默认模板包含 Shadowsocks 和 VLESS 两个节点，VLESS 第二 TLS 入口为 8443；主入口端口、WS 路径由面板下发。只需 VLESS 时可改用 config_sspanel_metron_ws_tls.yml.example。

配置存放在源码目录之外，不提交到 Git，也不写进镜像。`.dockerignore` 限定了构建上下文，
不包含 Git 历史、本地配置或证书文件。

## 证书和网络

将已有证书放入 `/opt/xrayr/config/cert/`，配置中的路径使用容器路径：

```yaml
CertConfig:
  CertMode: file
  CertDomain: node.example.com
  CertFile: /etc/XrayR/cert/fullchain.pem
  KeyFile: /etc/XrayR/cert/privkey.pem
```

宿主机的配置目录挂载到容器 `/etc/XrayR`。自动申请证书时改用相应 CertMode，
保持目录可写；容器工作目录也设为该目录，使证书状态随目录持久保存。

按实际节点配置开放防火墙：WS 需要 TCP 80，WSS / AnyTLS 通常需要 TCP 443，
Hysteria2 需要配置端口的 UDP。TCP 与 UDP 是不同端口类型，可使用相同数字；
两个 TCP 服务不能同时占用同一监听地址和端口。
容器使用 host 网络时不添加 `ports`，也不需要 `privileged`。

## 构建和启动

```bash
docker compose build
docker compose up -d
docker compose logs --tail=100 -f
```

默认挂载 `/opt/xrayr/config`。使用其他目录时：

```bash
XRAYR_CONFIG_DIR=/absolute/path/to/config docker compose up -d
```

Compose 要求该目录已经存在，避免把错误路径自动创建为空目录。
宿主机若已经运行 systemd 版 XrayR，请先停掉它，避免端口冲突。

## 升级与停止

```bash
git pull --ff-only
docker compose build --pull
docker compose up -d
docker compose logs --tail=100
```

配置与证书保存在宿主机挂载目录中，重建镜像不会覆盖它们。
停止容器用 `docker compose down`，不会删除上述宿主机目录。

## 不使用 Compose

```bash
docker build -t xrayr-local:1.0.0 .
docker run -d --name xrayr --restart unless-stopped --network host \
  --mount type=bind,source=/opt/xrayr/config,target=/etc/XrayR \
  xrayr-local:1.0.0
```

默认教程面向 Linux。Docker Desktop 的 host 网络支持条件不同，请按 Docker 官方说明配置。
本教程不假定本仓库的 GHCR 镜像已经发布成功。
