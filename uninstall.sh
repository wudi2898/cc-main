#!/bin/bash

# CC压测工具卸载脚本
# 完全清理安装的文件、服务、用户和配置

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置
PROJECT_NAME="cc-main"
SERVICE_NAME="cc-main"
INSTALL_DIR="/opt/cc-main"
SERVICE_USER="cc-main"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo -e "${PURPLE}================================${NC}"
    echo -e "${PURPLE}$1${NC}"
    echo -e "${PURPLE}================================${NC}"
}

# 检查是否为root用户
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "此脚本需要root权限运行"
        log_info "请使用: sudo $0"
        exit 1
    fi
}

# 检查操作系统
check_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
    else
        OS="unknown"
    fi
    
    log_info "检测到操作系统: $OS"
}

# 停止并删除服务
remove_service() {
    log_info "停止并删除系统服务..."
    
    if [ "$OS" = "linux" ]; then
        # 停止服务
        if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
            log_info "停止systemd服务..."
            systemctl stop "$SERVICE_NAME"
        fi
        
        # 禁用服务
        if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
            log_info "禁用systemd服务..."
            systemctl disable "$SERVICE_NAME"
        fi
        
        # 删除服务文件
        if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
            log_info "删除systemd服务文件..."
            rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
            systemctl daemon-reload
        fi
        
    elif [ "$OS" = "macos" ]; then
        # 停止launchd服务
        if launchctl list | grep -q "com.ccmain.webpanel" 2>/dev/null; then
            log_info "停止launchd服务..."
            launchctl stop com.ccmain.webpanel
            launchctl unload "/Library/LaunchDaemons/com.ccmain.webpanel.plist" 2>/dev/null || true
        fi
        
        # 删除服务文件
        if [ -f "/Library/LaunchDaemons/com.ccmain.webpanel.plist" ]; then
            log_info "删除launchd服务文件..."
            rm -f "/Library/LaunchDaemons/com.ccmain.webpanel.plist"
        fi
    fi
    
    log_success "系统服务已删除"
}

# 删除管理脚本
remove_management_scripts() {
    log_info "删除管理脚本..."
    
    local scripts=("cc-start" "cc-stop" "cc-status" "cc-restart" "cc-logs")
    
    for script in "${scripts[@]}"; do
        if [ -f "/usr/local/bin/$script" ]; then
            log_info "删除 $script..."
            rm -f "/usr/local/bin/$script"
        fi
    done
    
    log_success "管理脚本已删除"
}

# 删除用户和目录
remove_user_and_dirs() {
    log_info "删除用户和目录..."
    
    # 停止相关进程
    local pids=$(pgrep -f "web_panel.py" 2>/dev/null || echo "")
    if [ -n "$pids" ]; then
        log_info "停止相关进程..."
        echo "$pids" | xargs kill -TERM 2>/dev/null || true
        sleep 2
        echo "$pids" | xargs kill -KILL 2>/dev/null || true
    fi
    
    # 删除安装目录
    if [ -d "$INSTALL_DIR" ]; then
        log_info "删除安装目录: $INSTALL_DIR"
        rm -rf "$INSTALL_DIR"
    fi
    
    # 删除用户
    if id "$SERVICE_USER" &>/dev/null; then
        log_info "删除用户: $SERVICE_USER"
        userdel -r "$SERVICE_USER" 2>/dev/null || true
    fi
    
    log_success "用户和目录已删除"
}

# 清理系统配置
cleanup_system_config() {
    log_info "清理系统配置..."
    
    if [ "$OS" = "linux" ]; then
        # 恢复sysctl配置
        if [ -f "/etc/sysctl.conf" ]; then
            log_info "恢复sysctl配置..."
            # 备份原文件
            cp /etc/sysctl.conf /etc/sysctl.conf.backup.$(date +%Y%m%d_%H%M%S)
            
            # 删除CC压测工具相关配置
            sed -i '/# CC压测工具网络优化/,/^$/d' /etc/sysctl.conf
            sysctl -p 2>/dev/null || true
        fi
        
        # 恢复limits配置
        if [ -f "/etc/security/limits.conf" ]; then
            log_info "恢复limits配置..."
            # 备份原文件
            cp /etc/security/limits.conf /etc/security/limits.conf.backup.$(date +%Y%m%d_%H%M%S)
            
            # 删除CC压测工具相关配置
            sed -i '/# CC压测工具文件描述符限制/,/^$/d' /etc/security/limits.conf
        fi
    fi
    
    log_success "系统配置已清理"
}

# 清理日志文件
cleanup_logs() {
    log_info "清理日志文件..."
    
    # 清理systemd日志
    if [ "$OS" = "linux" ] && command -v journalctl &> /dev/null; then
        log_info "清理systemd日志..."
        journalctl --vacuum-time=1s --quiet 2>/dev/null || true
    fi
    
    # 清理其他日志
    local log_dirs=("/var/log" "/tmp" "/var/tmp")
    for dir in "${log_dirs[@]}"; do
        if [ -d "$dir" ]; then
            find "$dir" -name "*cc-main*" -type f -delete 2>/dev/null || true
            find "$dir" -name "*web_panel*" -type f -delete 2>/dev/null || true
        fi
    done
    
    log_success "日志文件已清理"
}

# 确认卸载
confirm_uninstall() {
    echo
    log_header "⚠️  确认卸载"
    echo
    echo -e "${YELLOW}此操作将完全删除CC压测工具，包括:${NC}"
    echo -e "  • 所有程序文件"
    echo -e "  • 系统服务配置"
    echo -e "  • 用户账户"
    echo -e "  • 日志文件"
    echo -e "  • 配置文件"
    echo
    echo -e "${RED}此操作不可逆！${NC}"
    echo
    
    read -p "$(echo -e ${CYAN}确定要继续卸载吗？ [y/N]: ${NC})" -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "卸载已取消"
        exit 0
    fi
}

# 显示卸载结果
show_uninstall_result() {
    echo
    log_header "✅ 卸载完成"
    echo
    echo -e "${GREEN}CC压测工具已完全卸载${NC}"
    echo
    echo -e "${CYAN}已删除的内容:${NC}"
    echo -e "  • 安装目录: $INSTALL_DIR"
    echo -e "  • 系统服务: $SERVICE_NAME"
    echo -e "  • 用户账户: $SERVICE_USER"
    echo -e "  • 管理脚本: /usr/local/bin/cc-*"
    echo -e "  • 日志文件: 相关日志已清理"
    echo
    echo -e "${YELLOW}注意:${NC}"
    echo -e "  • 系统配置文件已备份"
    echo -e "  • 如需恢复，请查看备份文件"
    echo
}

# 显示横幅
show_banner() {
    clear
    echo -e "${PURPLE}"
    echo "██████╗ ██████╗  ██████╗ ███████╗    ████████╗ ██████╗  ██████╗ ██╗     "
    echo "██╔══██╗██╔══██╗██╔═══██╗██╔════╝    ╚══██╔══╝██╔═══██╗██╔═══██╗██║     "
    echo "██║  ██║██║  ██║██║   ██║███████╗       ██║   ██║   ██║██║   ██║██║     "
    echo "██║  ██║██║  ██║██║   ██║╚════██║       ██║   ██║   ██║██║   ██║██║     "
    echo "██████╔╝██████╔╝╚██████╔╝███████║       ██║   ╚██████╔╝╚██████╔╝███████╗"
    echo "╚═════╝ ╚═════╝  ╚═════╝ ╚══════╝       ╚═╝    ╚═════╝  ╚═════╝ ╚══════╝"
    echo -e "${NC}"
    echo -e "${CYAN}          DDoS压力测试工具 - 卸载脚本 v3.0.0${NC}"
    echo -e "${YELLOW}              完全清理所有安装的文件和配置${NC}"
    echo
}

# 主函数
main() {
    show_banner
    
    # 检查环境
    check_root
    check_os
    
    # 确认卸载
    confirm_uninstall
    
    log_header "开始卸载CC压测工具"
    
    # 执行卸载步骤
    remove_service
    remove_management_scripts
    remove_user_and_dirs
    cleanup_system_config
    cleanup_logs
    
    # 显示结果
    show_uninstall_result
}

# 错误处理
trap 'log_error "卸载过程中出现错误！"; exit 1' ERR

# 运行主函数
main "$@"
