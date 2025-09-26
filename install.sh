#!/bin/bash

# 原始安装脚本 - 不包含性能优化
# 仅用于one_click_all.sh内部调用

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# 检查root权限
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "此脚本需要root权限运行"
        exit 1
    fi
}

# 安装系统依赖
install_system_deps() {
    log_info "安装系统依赖..."
    
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if [ -f /etc/debian_version ]; then
            DISTRO="debian"
        elif [ -f /etc/redhat-release ]; then
            DISTRO="redhat"
        else
            DISTRO="unknown"
        fi
    else
        DISTRO="unknown"
    fi
    
    if [ "$DISTRO" = "debian" ]; then
        apt update
        apt install -y python3 python3-pip python3-venv curl wget git
    elif [ "$DISTRO" = "redhat" ]; then
        yum update -y
        yum install -y python3 python3-pip curl wget git
    else
        log_warning "未识别的操作系统，尝试使用通用方法"
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
# SOCKS5代理列表 - 请添加真实代理
# 格式: IP:端口
# 示例:
# 127.0.0.1:1080
# 192.168.1.100:7890
# proxy.example.com:1080

# 临时测试代理（请替换为真实代理）
127.0.0.1:1080
EOF
        log_warning "请编辑 $INSTALL_DIR/config/socks5.txt 添加真实的SOCKS5代理"
    fi
    
    # 创建HTTP代理文件
    if [ ! -f "$INSTALL_DIR/config/http_proxies.txt" ] || [ ! -s "$INSTALL_DIR/config/http_proxies.txt" ]; then
        cat > "$INSTALL_DIR/config/http_proxies.txt" << EOF
# HTTP代理列表 - 请添加真实代理
# 格式: IP:端口
# 示例:
# 127.0.0.1:8080
# 192.168.1.100:3128
# proxy.example.com:8080
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
    log_info "创建systemd服务..."
    
    cat > "/etc/systemd/system/${SERVICE_NAME}.service" << EOF
[Unit]
Description=CC压测工具Web控制面板
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
Restart=on-failure
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

# 主函数
main() {
    log_info "开始安装CC压测工具..."
    
    # 检查权限
    check_root
    
    # 安装和配置
    install_system_deps
    setup_user_and_dirs
    install_python_deps
    create_config
    create_systemd_service
    create_management_scripts
    
    log_success "CC压测工具安装完成"
}

# 运行主函数
main "$@"