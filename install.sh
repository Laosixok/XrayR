#!/usr/bin/env bash
# Install or upgrade the binaries published by Laosixok/XrayR.
set -euo pipefail

repo="Laosixok/XrayR"
install_dir="/usr/local/XrayR"
config_dir="/etc/XrayR"
unit_dir="/etc/systemd/system"
bin_link="/usr/local/bin/XrayR"
version="latest"
template="default"
restart_running=1
work_dir=""
menu_only=0

fail() { printf '错误：%s\n' "$*" >&2; exit 1; }
info() { printf '%s\n' "$*"; }
usage() {
    cat <<'HELP'
用法：bash install.sh [--version 1.0.0] [--template default|ws-tls|protocols] [--no-restart]
默认下载本仓库最新 Release；重复运行用于升级。
现有配置、证书和 systemd 单元不覆盖。首次安装不自动启动。
--menu-only 只安装或修复管理菜单，不重新下载主程序。
--template 只决定首次创建的配置模板；不会替换现有 config.yml。
--no-restart 升级后不重启已运行的服务。
HELP
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version) [[ $# -ge 2 ]] || fail '--version 缺少版本号'; version="$2"; shift 2 ;;
            --template) [[ $# -ge 2 ]] || fail '--template 缺少模板名'; template="$2"; shift 2 ;;
            --menu-only) menu_only=1; shift ;;
            --no-restart) restart_running=0; shift ;;
            -h|--help) usage; exit 0 ;;
            *) fail "未知参数：$1" ;;
        esac
    done
    [[ "$version" =~ ^[a-zA-Z0-9][a-zA-Z0-9._-]*$ ]] || fail '版本号格式无效'
    case "$template" in default|ws-tls|protocols) ;; *) fail '模板仅支持 default、ws-tls 或 protocols' ;; esac
}

require_host() {
    [[ "$(uname -s)" == Linux ]] || fail '仅支持 Linux systemd 主机'
    [[ "$(id -u)" == 0 ]] || fail '请使用 root 或 sudo bash install.sh'
    case "$(uname -m)" in
        x86_64|amd64) asset="XrayR-linux-amd64.zip" ;;
        *) fail '当前发行版仅提供 Linux amd64 程序；其他架构请从源码构建' ;;
    esac
    local cmd
    for cmd in curl unzip sha256sum awk install mktemp systemctl; do
        command -v "$cmd" >/dev/null 2>&1 || fail "缺少 $cmd，请先安装 curl、unzip、ca-certificates、coreutils 和 systemd"
    done
    systemctl --version >/dev/null || fail 'systemd 不可用；容器请使用 Docker 部署方式'
}

resolve_version() {
    if [[ "$version" == latest ]]; then
        local resolved
        resolved=$(curl -fsSL --retry 3 -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
        version="${resolved##*/}"
    fi
    [[ "$version" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+([._-][a-zA-Z0-9._-]+)?$ ]] || fail '无法解析发行版本，请使用 --version 指定版本'
}

verify_archive() {
    local expected actual name
    expected=$(awk -v name="$asset" '$2 == name || $2 == "*" name {print $1}' "$work_dir/SHA256SUMS")
    [[ "$expected" =~ ^[a-fA-F0-9]{64}$ ]] || fail '校验文件中没有唯一有效的程序摘要'
    actual=$(sha256sum "$work_dir/$asset" | awk '{print $1}')
    [[ "$actual" == "$expected" ]] || fail '程序校验失败，未修改现有安装'
    while IFS= read -r name; do
        case "$name" in /*|*\\*|..|../*|*/../*|*/..) fail '压缩包包含不安全路径' ;; esac
    done < <(unzip -Z1 "$work_dir/$asset")
}

install_release() {
    local was_running=0 file candidate had_binary=0
    work_dir=$(mktemp -d)
    trap '[[ -z "$work_dir" ]] || rm -rf -- "$work_dir"' EXIT
    local base="https://github.com/$repo/releases/download/$version"
    info "下载 XrayR $version ($asset)…"
    curl -fsSL --retry 3 "$base/$asset" -o "$work_dir/$asset"
    curl -fsSL --retry 3 "$base/SHA256SUMS" -o "$work_dir/SHA256SUMS"
    verify_archive
    download_manager
    mkdir "$work_dir/unpacked"
    unzip -q "$work_dir/$asset" -d "$work_dir/unpacked"
    [[ -f "$work_dir/unpacked/XrayR" && ! -L "$work_dir/unpacked/XrayR" ]] || fail '发行包缺少常规程序文件'
    chmod 0755 "$work_dir/unpacked/XrayR"
    "$work_dir/unpacked/XrayR" version || fail '新程序无法运行，未修改现有安装'
    case "$template" in
        default) candidate="config_sspanel_metron_default.yml.example" ;;
        ws-tls) candidate="config_sspanel_metron_ws_tls.yml.example" ;;
        protocols) candidate="config_sspanel_metron_protocols.yml.example" ;;
    esac
    if [[ "$template" == default && ! -f "$work_dir/unpacked/$candidate" && ! -e "$config_dir/config.yml" ]]; then
        curl -fsSL --retry 3 "https://raw.githubusercontent.com/$repo/master/release/config/$candidate" -o "$work_dir/unpacked/$candidate"
    fi
    [[ -e "$config_dir/config.yml" || -f "$work_dir/unpacked/$candidate" ]] || fail '发行包缺少所选配置模板'
    if systemctl is-active --quiet XrayR.service; then was_running=1; fi
    install -d -m 0755 "$install_dir" "$unit_dir" "$(dirname "$bin_link")"
    install -d -m 0700 "$config_dir"
    if [[ -f "$install_dir/XrayR" ]]; then
        had_binary=1
        install -m 0755 "$install_dir/XrayR" "$install_dir/XrayR.previous"
    fi
    install -m 0755 "$work_dir/unpacked/XrayR" "$install_dir/XrayR.new"
    mv -f "$install_dir/XrayR.new" "$install_dir/XrayR"
    install_manager
    if [[ ! -e "$config_dir/config.yml" ]]; then
        install -m 0600 "$work_dir/unpacked/$candidate" "$config_dir/config.yml"
    fi
    for file in dns.json route.json custom_inbound.json custom_outbound.json rulelist geoip.dat geosite.dat; do
        if [[ ! -e "$config_dir/$file" && -f "$work_dir/unpacked/$file" ]]; then
            install -m 0600 "$work_dir/unpacked/$file" "$config_dir/$file"
        fi
    done
    for file in LICENSE NOTICE.md THIRD_PARTY_NOTICES.md; do
        [[ ! -f "$work_dir/unpacked/$file" ]] || install -m 0644 "$work_dir/unpacked/$file" "$install_dir/$file"
    done
    if [[ ! -e "$unit_dir/XrayR.service" ]]; then
        cat > "$unit_dir/XrayR.service" <<UNIT
[Unit]
Description=XrayR backend for SSPanel-Metron
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
WorkingDirectory=$config_dir
ExecStart=$install_dir/XrayR --config $config_dir/config.yml
Restart=on-failure
RestartSec=5s
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
UNIT
    fi
    systemctl daemon-reload
    if [[ "$was_running" == 1 && "$restart_running" == 1 ]]; then
        if ! systemctl restart XrayR.service; then
            if [[ "$had_binary" == 1 ]]; then
                mv -f "$install_dir/XrayR.previous" "$install_dir/XrayR"
                systemctl restart XrayR.service || true
            fi
            fail '新版本重启失败，已尝试恢复旧程序；请检查 journalctl -u XrayR'
        fi
    fi
    info "XrayR $version 安装完成；现有配置和证书保持不变。"
    info "首次安装请编辑 $config_dir/config.yml，填写面板信息和证书，再运行："
    info 'systemctl enable --now XrayR.service'
    info '查看日志：journalctl -u XrayR.service -f'
}

download_manager() {
    curl -fsSL --retry 3 "https://raw.githubusercontent.com/$repo/master/XrayR.sh" -o "$work_dir/manager.sh"
    bash -n "$work_dir/manager.sh" || fail '管理脚本语法检查失败'
}
install_manager() {
    install -d -m 0755 "$install_dir" "$(dirname "$bin_link")"
    install -m 0755 "$work_dir/manager.sh" "$install_dir/manager.sh.new"
    mv -f "$install_dir/manager.sh.new" "$install_dir/manager.sh"
    ln -sfn "$install_dir/manager.sh" "$bin_link"
    ln -sfn "$install_dir/manager.sh" "$(dirname "$bin_link")/xrayr"
    info '管理菜单已安装，执行 xrayr 或 XrayR 打开。'
}
main() {
    parse_args "$@"; require_host
    if [[ "$menu_only" == 1 ]]; then
        [[ -x "$install_dir/XrayR" ]] || fail '未找到已安装的主程序，请先正常安装'
        work_dir=$(mktemp -d)
        trap '[[ -z "$work_dir" ]] || rm -rf -- "$work_dir"' EXIT
        download_manager; install_manager
    else resolve_version; install_release; fi
}
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then main "$@"; fi
