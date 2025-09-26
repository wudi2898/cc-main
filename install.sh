#!/bin/bash

# DDoS压测工具一键安装脚本
# 支持自动依赖安装、配置检查、启动服务、开机自启

set -e  # 出错时退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 项目配置
PROJECT_NAME="cc-main"
SERVICE_NAME="cc-main"
INSTALL_DIR="/opt/cc-main"
SERVICE_USER="cc-main"
WEB_PORT="5000"

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
        if [ -f /etc/debian_version ]; then
            DISTRO="debian"
        elif [ -f /etc/redhat-release ]; then
            DISTRO="redhat"
        elif [ -f /etc/arch-release ]; then
            DISTRO="arch"
        else
            DISTRO="unknown"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        DISTRO="macOS"
    else
        OS="unknown"
        DISTRO="Unknown"
    fi
    
    log_info "检测到操作系统: $DISTRO ($OS)"
    
    if [ "$OS" = "unknown" ]; then
        log_error "不支持的操作系统"
        exit 1
    fi
}

# 安装系统依赖
install_system_deps() {
    log_info "安装系统依赖..."
    
    if [ "$OS" = "linux" ]; then
        case $DISTRO in
            "debian"|"ubuntu")
                apt update
                apt install -y python3 python3-pip python3-venv curl wget git
                ;;
            "redhat"|"centos"|"fedora")
                yum update -y
                yum install -y python3 python3-pip curl wget git
                ;;
            "arch")
                pacman -Syu --noconfirm
                pacman -S --noconfirm python python-pip curl wget git
                ;;
        esac
    elif [ "$OS" = "macos" ]; then
        if ! command -v brew &> /dev/null; then
            log_info "安装Homebrew..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        fi
        brew install python3 curl wget git
    fi
    
    log_success "系统依赖安装完成"
}

# 创建用户和目录
setup_user_and_dirs() {
    log_info "创建用户和目录..."
    
    # 创建专用用户
    if ! id "$SERVICE_USER" &>/dev/null; then
        useradd -r -s /bin/false -d "$INSTALL_DIR" "$SERVICE_USER"
        log_success "创建用户: $SERVICE_USER"
    else
        log_info "用户 $SERVICE_USER 已存在"
    fi
    
    # 创建安装目录
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR/logs"
    mkdir -p "$INSTALL_DIR/config"
    
    # 复制项目文件
    cp -r . "$INSTALL_DIR/"
    
    # 设置权限
    chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
    chmod +x "$INSTALL_DIR"/*.py
    chmod +x "$INSTALL_DIR"/*.sh
    
    log_success "目录和权限设置完成"
}

# 安装Python依赖
install_python_deps() {
    log_info "安装Python依赖..."
    
    cd "$INSTALL_DIR"
    
    # 创建虚拟环境
    python3 -m venv venv
    source venv/bin/activate
    
    # 升级pip
    pip install --upgrade pip
    
    # 安装依赖
    pip install -r requirements.txt
    
    # 验证安装
    python -c "import flask, flask_socketio, psutil, socks" && {
        log_success "Python依赖安装成功"
    } || {
        log_error "Python依赖安装失败"
        exit 1
    }
}

# 创建配置文件
create_config() {
    log_info "创建配置文件..."
    
    # 创建配置目录
    mkdir -p "$INSTALL_DIR/config"
    
    # 创建示例代理文件
    if [ ! -f "$INSTALL_DIR/config/socks5.txt" ] || [ ! -s "$INSTALL_DIR/config/socks5.txt" ]; then
        cat > "$INSTALL_DIR/config/socks5.txt" << EOF
# SOCKS5代理列表 - 每行一个代理
# 格式: IP:端口
# 示例:
# 127.0.0.1:1080
# 192.168.1.100:7890
EOF
        log_warning "请编辑 $INSTALL_DIR/config/socks5.txt 添加真实的SOCKS5代理"
    fi
    
    # 创建HTTP代理文件
    if [ ! -f "$INSTALL_DIR/config/http_proxies.txt" ] || [ ! -s "$INSTALL_DIR/config/http_proxies.txt" ]; then
        cat > "$INSTALL_DIR/config/http_proxies.txt" << EOF
# HTTP代理列表 - 每行一个代理
# 格式: IP:端口
# 示例:
# 127.0.0.1:8080
# 192.168.1.100:3128
EOF
    fi
    
    # 创建系统配置文件
    cat > "$INSTALL_DIR/config/system.conf" << EOF
# 系统配置文件
WEB_PORT=$WEB_PORT
LOG_LEVEL=INFO
MAX_CONNECTIONS=1000
DEFAULT_THREADS=100
DEFAULT_RPS=10
EOF
    
    chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
    log_success "配置文件创建完成"
}

# 创建系统服务
create_systemd_service() {
    if [ "$OS" = "linux" ]; then
        log_info "创建systemd服务..."
        
        cat > "/etc/systemd/system/${SERVICE_NAME}.service" << EOF
[Unit]
Description=DDoS压测工具Web控制面板
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$INSTALL_DIR
Environment=PATH=$INSTALL_DIR/venv/bin
ExecStart=$INSTALL_DIR/venv/bin/python $INSTALL_DIR/web_panel.py
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$INSTALL_DIR

[Install]
WantedBy=multi-user.target
EOF
        
        # 重载systemd并启用服务
        systemctl daemon-reload
        systemctl enable "$SERVICE_NAME"
        
        log_success "systemd服务创建完成"
    elif [ "$OS" = "macos" ]; then
        log_info "创建launchd服务..."
        
        cat > "/Library/LaunchDaemons/com.ccmain.webpanel.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.ccmain.webpanel</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/venv/bin/python</string>
        <string>$INSTALL_DIR/web_panel.py</string>
    </array>
    <key>WorkingDirectory</key>
    <string>$INSTALL_DIR</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$INSTALL_DIR/logs/webpanel.log</string>
    <key>StandardErrorPath</key>
    <string>$INSTALL_DIR/logs/webpanel.error.log</string>
    <key>UserName</key>
    <string>$SERVICE_USER</string>
</dict>
</plist>
EOF
        
        # 设置权限并加载服务
        chown root:wheel "/Library/LaunchDaemons/com.ccmain.webpanel.plist"
        chmod 644 "/Library/LaunchDaemons/com.ccmain.webpanel.plist"
        launchctl load "/Library/LaunchDaemons/com.ccmain.webpanel.plist"
        
        log_success "launchd服务创建完成"
    fi
}

# 创建管理脚本
create_management_scripts() {
    log_info "创建管理脚本..."
    
    # 启动脚本
    cat > "/usr/local/bin/cc-start" << 'EOF'
#!/bin/bash
# 启动CC压测工具

if [ "$EUID" -ne 0 ]; then
    echo "需要root权限运行"
    exit 1
fi

systemctl start cc-main
echo "CC压测工具已启动"
echo "Web面板: http://localhost:5000"
EOF

    # 停止脚本
    cat > "/usr/local/bin/cc-stop" << 'EOF'
#!/bin/bash
# 停止CC压测工具

if [ "$EUID" -ne 0 ]; then
    echo "需要root权限运行"
    exit 1
fi

systemctl stop cc-main
echo "CC压测工具已停止"
EOF

    # 状态脚本
    cat > "/usr/local/bin/cc-status" << 'EOF'
#!/bin/bash
# 查看CC压测工具状态

if [ "$EUID" -ne 0 ]; then
    echo "需要root权限运行"
    exit 1
fi

systemctl status cc-main
EOF

    # 重启脚本
    cat > "/usr/local/bin/cc-restart" << 'EOF'
#!/bin/bash
# 重启CC压测工具

if [ "$EUID" -ne 0 ]; then
    echo "需要root权限运行"
    exit 1
fi

systemctl restart cc-main
echo "CC压测工具已重启"
EOF

    # 日志脚本
    cat > "/usr/local/bin/cc-logs" << 'EOF'
#!/bin/bash
# 查看CC压测工具日志

if [ "$EUID" -ne 0 ]; then
    echo "需要root权限运行"
    exit 1
fi

journalctl -u cc-main -f
EOF

    # 设置执行权限
    chmod +x /usr/local/bin/cc-*

    log_success "管理脚本创建完成"
}

# 系统优化
optimize_system() {
    log_info "应用系统优化..."
    
    if [ "$OS" = "linux" ]; then
        # 网络优化
        cat >> /etc/sysctl.conf << EOF

# CC压测工具网络优化
net.core.somaxconn = 65535
net.ipv4.ip_local_port_range = 1024 65535
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_keepalive_time = 1200
net.ipv4.tcp_keepalive_intvl = 15
net.ipv4.tcp_keepalive_probes = 5
EOF
        
        # 应用设置
        sysctl -p
        
        # 文件描述符限制
        cat >> /etc/security/limits.conf << EOF

# CC压测工具文件描述符限制
$SERVICE_USER soft nofile 65535
$SERVICE_USER hard nofile 65535
root soft nofile 65535
root hard nofile 65535
EOF
        
        log_success "系统优化完成"
    else
        log_info "非Linux系统，跳过系统优化"
    fi
}

# 启动服务
start_service() {
    log_info "启动服务..."
    
    if [ "$OS" = "linux" ]; then
        systemctl start "$SERVICE_NAME"
        systemctl status "$SERVICE_NAME" --no-pager
    elif [ "$OS" = "macos" ]; then
        launchctl start com.ccmain.webpanel
    fi
    
    # 等待服务启动
    sleep 3
    
    # 检查服务状态
    if [ "$OS" = "linux" ]; then
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            log_success "服务启动成功"
        else
            log_error "服务启动失败"
            systemctl status "$SERVICE_NAME" --no-pager
            exit 1
        fi
    fi
}

# 显示安装结果
show_installation_result() {
    local_ip=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "127.0.0.1")
    
    echo
    log_header "🎉 安装完成！"
    echo
    echo -e "${CYAN}📁 安装目录:${NC} $INSTALL_DIR"
    echo -e "${CYAN}👤 运行用户:${NC} $SERVICE_USER"
    echo -e "${CYAN}🌐 Web面板:${NC} http://localhost:$WEB_PORT"
    echo -e "${CYAN}🌍 远程访问:${NC} http://$local_ip:$WEB_PORT"
    echo
    echo -e "${CYAN}📋 管理命令:${NC}"
    echo -e "  ${GREEN}cc-start${NC}    - 启动服务"
    echo -e "  ${GREEN}cc-stop${NC}     - 停止服务"
    echo -e "  ${GREEN}cc-restart${NC}  - 重启服务"
    echo -e "  ${GREEN}cc-status${NC}   - 查看状态"
    echo -e "  ${GREEN}cc-logs${NC}     - 查看日志"
    echo
    echo -e "${CYAN}📝 配置文件:${NC}"
    echo -e "  ${YELLOW}代理列表:${NC} $INSTALL_DIR/config/socks5.txt"
    echo -e "  ${YELLOW}HTTP代理:${NC} $INSTALL_DIR/config/http_proxies.txt"
    echo -e "  ${YELLOW}系统配置:${NC} $INSTALL_DIR/config/system.conf"
    echo -e "  ${YELLOW}日志目录:${NC} $INSTALL_DIR/logs/"
    echo
    echo -e "${CYAN}🚀 开机自启:${NC} 已启用"
    echo -e "${CYAN}📊 服务状态:${NC} 运行中"
    echo
    echo -e "${YELLOW}⚠️  重要提醒:${NC}"
    echo -e "  • 请编辑 $INSTALL_DIR/config/socks5.txt 添加真实代理"
    echo -e "  • 仅用于授权的安全测试"
    echo -e "  • 遵守当地法律法规"
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
    echo -e "${CYAN}          DDoS压力测试工具 - 一键安装脚本 v3.0.0${NC}"
    echo -e "${YELLOW}              支持自动安装、配置、启动、开机自启${NC}"
    echo
}

# 主函数
main() {
    show_banner
    
    log_header "开始安装DDoS压测工具"
    
    # 检查环境
    check_root
    check_os
    
    # 安装和配置
    install_system_deps
    setup_user_and_dirs
    install_python_deps
    create_config
    create_systemd_service
    create_management_scripts
    
    # 系统优化
    read -p "$(echo -e ${CYAN}是否进行系统网络优化？ [y/N]: ${NC})" -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        optimize_system
    fi
    
    # 启动服务
    start_service
    
    # 显示结果
    show_installation_result
}

# 错误处理
trap 'log_error "安装过程中出现错误！"; exit 1' ERR

# 运行主函数
main "$@"