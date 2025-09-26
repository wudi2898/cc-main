#!/bin/bash

# 修复服务重复执行问题

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# 检查root权限
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "此脚本需要root权限运行"
        log_info "请使用: sudo $0"
        exit 1
    fi
}

# 停止所有相关进程
stop_all_processes() {
    log_info "停止所有相关进程..."
    
    # 停止systemd服务
    if systemctl is-active --quiet cc-main 2>/dev/null; then
        systemctl stop cc-main
        log_info "已停止systemd服务"
    fi
    
    # 杀死所有web_panel.py进程
    pkill -f "web_panel.py" 2>/dev/null || true
    log_info "已停止所有web_panel.py进程"
    
    # 杀死所有main.py进程
    pkill -f "main.py" 2>/dev/null || true
    log_info "已停止所有main.py进程"
    
    # 等待进程完全停止
    sleep 3
    
    log_success "所有进程已停止"
}

# 检查进程状态
check_processes() {
    log_info "检查进程状态..."
    
    local web_processes=$(pgrep -f "web_panel.py" | wc -l)
    local main_processes=$(pgrep -f "main.py" | wc -l)
    
    echo "Web面板进程数: $web_processes"
    echo "主程序进程数: $main_processes"
    
    if [ "$web_processes" -gt 0 ]; then
        log_warning "发现 $web_processes 个web_panel.py进程"
        pgrep -f "web_panel.py" | xargs ps -p
    fi
    
    if [ "$main_processes" -gt 0 ]; then
        log_warning "发现 $main_processes 个main.py进程"
        pgrep -f "main.py" | xargs ps -p
    fi
}

# 清理systemd服务
cleanup_systemd() {
    log_info "清理systemd服务..."
    
    # 停止并禁用服务
    systemctl stop cc-main 2>/dev/null || true
    systemctl disable cc-main 2>/dev/null || true
    
    # 删除服务文件
    rm -f /etc/systemd/system/cc-main.service
    
    # 重载systemd
    systemctl daemon-reload
    
    log_success "systemd服务已清理"
}

# 重新创建正确的服务
recreate_service() {
    log_info "重新创建服务..."
    
    # 创建正确的服务文件
    cat > /etc/systemd/system/cc-main.service << 'EOF'
[Unit]
Description=CC压测工具Web控制面板
After=network.target
Wants=network.target

[Service]
Type=simple
User=cc-main
Group=cc-main
WorkingDirectory=/opt/cc-main
Environment=PATH=/opt/cc-main/venv/bin
ExecStart=/opt/cc-main/venv/bin/python /opt/cc-main/web_panel.py
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=cc-main

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/cc-main

[Install]
WantedBy=multi-user.target
EOF
    
    # 重载systemd
    systemctl daemon-reload
    systemctl enable cc-main
    
    log_success "服务已重新创建"
}

# 启动服务
start_service() {
    log_info "启动服务..."
    
    # 启动服务
    systemctl start cc-main
    
    # 等待服务启动
    sleep 5
    
    # 检查服务状态
    if systemctl is-active --quiet cc-main; then
        log_success "服务启动成功"
        systemctl status cc-main --no-pager
    else
        log_error "服务启动失败"
        systemctl status cc-main --no-pager
        journalctl -u cc-main --no-pager -n 20
    fi
}

# 显示结果
show_result() {
    echo
    echo "=================================="
    echo "🔧 重复执行问题修复完成"
    echo "=================================="
    echo
    
    # 检查最终状态
    check_processes
    
    echo
    echo -e "${CYAN}📋 管理命令:${NC}"
    echo -e "  ${GREEN}cc-start${NC}    - 启动服务"
    echo -e "  ${GREEN}cc-stop${NC}     - 停止服务"
    echo -e "  ${GREEN}cc-restart${NC}  - 重启服务"
    echo -e "  ${GREEN}cc-status${NC}   - 查看状态"
    echo -e "  ${GREEN}cc-logs${NC}     - 查看日志"
    echo
    echo -e "${GREEN}✅ 问题已修复，服务现在应该正常运行了！${NC}"
    echo
}

# 主函数
main() {
    echo "🔧 修复服务重复执行问题"
    echo "========================"
    echo
    
    # 检查权限
    check_root
    
    # 停止所有进程
    stop_all_processes
    
    # 检查进程状态
    check_processes
    
    # 清理systemd服务
    cleanup_systemd
    
    # 重新创建服务
    recreate_service
    
    # 启动服务
    start_service
    
    # 显示结果
    show_result
}

# 运行主函数
main "$@"
