#!/usr/bin/env bash
# XrayR systemd management entry; the daemon remains /usr/local/XrayR/XrayR.
set -euo pipefail
repo="Laosixok/XrayR"
install_dir="/usr/local/XrayR"
config_dir="/etc/XrayR"
unit_dir="/etc/systemd/system"

require_root() { [[ "$(id -u)" == 0 ]] || { echo '请使用 sudo xrayr'; return 1; }; }
run_installer() {
    local installer result=0
    installer=$(mktemp)
    if curl -fsSL --retry 3 "https://raw.githubusercontent.com/$repo/master/install.sh" -o "$installer" && bash -n "$installer"; then
        bash "$installer" "$@" || result=$?
    else result=1; fi
    rm -f "$installer"
    return "$result"
}
edit_config() {
    [[ -f "$config_dir/config.yml" ]] || { echo '尚未安装配置，请先安装。'; return 1; }
    local editor="${EDITOR:-}"
    if [[ -z "$editor" ]]; then
        if command -v nano >/dev/null; then editor=nano; else editor=vi; fi
    fi
    "$editor" "$config_dir/config.yml"
    echo '配置已保存；需要生效时执行 xrayr restart。'
}
uninstall_service() {
    local answer
    printf '确认卸载程序和服务？配置、证书保留。输入 UNINSTALL 确认：'
    read -r answer || return 1
    [[ "$answer" == UNINSTALL ]] || { echo '已取消。'; return 0; }
    systemctl disable --now XrayR.service || true
    rm -f "$unit_dir/XrayR.service" "$install_dir/XrayR" "$install_dir/XrayR.previous" "$install_dir/manager.sh"
    local link
    for link in /usr/local/bin/XrayR /usr/local/bin/xrayr; do
        if [[ -L "$link" && "$(readlink "$link")" == "$install_dir/manager.sh" ]]; then rm -f "$link"; fi
    done
    systemctl daemon-reload
    echo '已卸载；配置和证书保持不变。'
}
reset_template() {
    local answer temporary
    printf '用最新默认模板替换配置？会先备份原配置。输入 RESET 确认：'
    read -r answer || return 1
    [[ "$answer" == RESET ]] || { echo '已取消。'; return 0; }
    temporary=$(mktemp)
    if ! curl -fsSL --retry 3 "https://raw.githubusercontent.com/$repo/master/release/config/config_sspanel_metron_default.yml.example" -o "$temporary"; then rm -f "$temporary"; return 1; fi
    install -d -m 0700 "$config_dir"
    if [[ -f "$config_dir/config.yml" ]]; then
        install -m 0600 "$config_dir/config.yml" "$config_dir/config.yml.backup.$(date +%Y%m%d%H%M%S)"
    fi
    install -m 0600 "$temporary" "$config_dir/config.yml"
    rm -f "$temporary"
    echo '默认模板已生成，请填写配置后自行重启。'
}
show_menu() {
    local choice running enabled
    while true; do
        running='未运行'; enabled='否'
        if systemctl is-active --quiet XrayR.service; then running='已运行'; fi
        if systemctl is-enabled --quiet XrayR.service; then enabled='是'; fi
        cat <<MENU

XrayR 后端管理脚本（systemd；Docker 请使用 docker compose）
https://github.com/$repo
 0. 修改配置
 1. 安装 XrayR
 2. 更新 XrayR
 3. 卸载 XrayR（保留配置）
 4. 启动 XrayR
 5. 停止 XrayR
 6. 重启 XrayR
 7. 查看状态
 8. 查看日志
 9. 设置开机自启
10. 取消开机自启
11. 查看程序版本
12. 更新管理脚本
13. 生成默认配置（先备份）
 q. 退出
XrayR 状态：${running}；开机自启：${enabled}
MENU
        printf '请选择 [0-13/q]：'
        read -r choice || return 0
        case "$choice" in
            0) dispatch config || true ;; 1) dispatch install || true ;;
            2) dispatch update || true ;; 3) dispatch uninstall || true ;;
            4) dispatch start || true ;; 5) dispatch stop || true ;;
            6) dispatch restart || true ;; 7) dispatch status || true ;;
            8) dispatch log || true ;; 9) dispatch enable || true ;;
            10) dispatch disable || true ;; 11) dispatch version || true ;;
            12) dispatch update-shell || true ;; 13) dispatch reset-config || true ;;
            q|Q) return 0 ;; *) echo '无效选项。' ;;
        esac
    done
}
dispatch() {
    local action="${1:-menu}"
    [[ $# == 0 ]] || shift
    case "$action" in
        menu) show_menu ;;
        install|update) if [[ $# -gt 0 && "$1" != --* ]]; then run_installer --version "$@"; else run_installer "$@"; fi ;;
        start|stop|restart|status|enable|disable) systemctl "$action" XrayR.service ;;
        log) journalctl -u XrayR.service -n 100 -f ;;
        config) edit_config ;;
        version) "$install_dir/XrayR" version ;;
        update-shell) run_installer --menu-only ;;
        uninstall) uninstall_service ;;
        reset-config) reset_template ;;
        -h|--help) echo 'xrayr [menu|install|update [版本]|start|stop|restart|status|log|enable|disable|config|version|update-shell|uninstall|reset-config]' ;;
        *) "$install_dir/XrayR" "$action" "$@" ;;
    esac
}
main() { require_root; dispatch "$@"; }
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then main "$@"; fi
