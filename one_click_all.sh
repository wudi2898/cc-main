#!/bin/bash
# CC压测工具 - 真正的一键安装脚本
# 支持Linux和macOS

set -e

# 错误处理函数
handle_error() {
    echo -e "${RED}❌ 安装过程中出现错误！${NC}"
    echo -e "${BLUE}请检查上面的错误信息，或尝试手动安装${NC}"
    exit 1
}

# 设置错误陷阱
trap 'handle_error' ERR

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🚀 CC压测工具 - 一键安装${NC}"
echo

# 检测操作系统
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
    echo -e "${BLUE}检测到Linux系统${NC}"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
    echo -e "${BLUE}检测到macOS系统${NC}"
else
    echo -e "${RED}不支持的操作系统${NC}"
    exit 1
fi

# 设置项目目录
if [ "$OS" = "linux" ]; then
    PROJECT_DIR="/opt/cc-main"
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}Linux需要root权限，请使用: sudo $0${NC}"
        exit 1
    fi
else
    PROJECT_DIR="$HOME/cc-main"
fi

echo -e "${BLUE}项目目录: $PROJECT_DIR${NC}"

# 创建项目目录
mkdir -p "$PROJECT_DIR"
cd "$PROJECT_DIR"

# 下载必要文件
echo -e "${BLUE}📥 下载项目文件...${NC}"
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/main.py -o main.py || {
    echo -e "${RED}下载main.py失败${NC}"
    exit 1
}
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/web_panel.py -o web_panel.py || {
    echo -e "${RED}下载web_panel.py失败${NC}"
    exit 1
}
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/requirements.txt -o requirements.txt || {
    echo -e "${RED}下载requirements.txt失败${NC}"
    exit 1
}

# 创建配置目录和文件
mkdir -p config
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/accept_headers.txt -o config/accept_headers.txt || {
    echo -e "${RED}下载accept_headers.txt失败${NC}"
    exit 1
}
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/referers.txt -o config/referers.txt || {
    echo -e "${RED}下载referers.txt失败${NC}"
    exit 1
}
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/socks5.txt -o config/socks5.txt || {
    echo -e "${RED}下载socks5.txt失败${NC}"
    exit 1
}
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/http_proxies.txt -o config/http_proxies.txt || {
    echo -e "${RED}下载http_proxies.txt失败${NC}"
    exit 1
}

# 创建templates目录和文件
mkdir -p templates
curl -fsSL https://raw.githubusercontent.com/wudi2898/cc-main/main/templates/index.html -o templates/index.html || {
    echo -e "${RED}下载index.html失败${NC}"
    exit 1
}

# 创建logs目录
mkdir -p logs

# 设置权限
chmod +x *.py 2>/dev/null || true

# 安装系统依赖
echo -e "${BLUE}📦 安装系统依赖...${NC}"
if [ "$OS" = "linux" ]; then
    if command -v apt-get &> /dev/null; then
        apt-get update
        apt-get install -y python3 python3-pip python3-venv curl wget
    elif command -v yum &> /dev/null; then
        yum update -y
        yum install -y python3 python3-pip curl wget
    elif command -v dnf &> /dev/null; then
        dnf update -y
        dnf install -y python3 python3-pip curl wget
    else
        echo -e "${BLUE}使用系统默认Python...${NC}"
    fi
elif [ "$OS" = "macos" ]; then
    if ! command -v brew &> /dev/null; then
        echo -e "${BLUE}安装Homebrew...${NC}"
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" || true
    fi
    if command -v brew &> /dev/null; then
        brew install python3 curl wget
    else
        echo -e "${BLUE}使用系统默认Python...${NC}"
    fi
fi

# 创建Python虚拟环境
echo -e "${BLUE}🐍 设置Python环境...${NC}"
python3 -m venv venv || {
    echo -e "${RED}创建虚拟环境失败，尝试使用系统Python...${NC}"
    # 如果venv失败，直接使用系统Python
    mkdir -p venv/bin
    ln -sf $(which python3) venv/bin/python
    ln -sf $(which pip3) venv/bin/pip
}

source venv/bin/activate

# 验证Python环境
if ! python --version &> /dev/null; then
    echo -e "${RED}Python环境验证失败${NC}"
    exit 1
fi

echo -e "${BLUE}Python版本: $(python --version)${NC}"

# 安装依赖
pip install --upgrade pip || pip3 install --upgrade pip
pip install -r requirements.txt || pip3 install -r requirements.txt

# 验证关键依赖
python -c "import flask, flask_socketio, psutil, socks" || {
    echo -e "${RED}依赖安装验证失败${NC}"
    exit 1
}

# 创建启动脚本
cat > start.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
source venv/bin/activate
python3 web_panel.py
EOF

chmod +x start.sh

# 创建停止脚本
cat > stop.sh << 'EOF'
#!/bin/bash
pkill -f "web_panel.py" 2>/dev/null || true
echo "服务已停止"
EOF

chmod +x stop.sh

# Linux系统创建服务
if [ "$OS" = "linux" ]; then
    echo -e "${BLUE}🔧 创建系统服务...${NC}"
    
    # 创建systemd服务
    cat > /etc/systemd/system/cc-main.service << EOF
[Unit]
Description=CC压测工具Web控制面板
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=$PROJECT_DIR
Environment=PATH=$PROJECT_DIR/venv/bin
ExecStart=$PROJECT_DIR/venv/bin/python $PROJECT_DIR/web_panel.py
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    # 启动服务
    systemctl daemon-reload
    systemctl enable cc-main
    systemctl start cc-main
    
    sleep 5
    
    if systemctl is-active --quiet cc-main; then
        echo -e "${GREEN}✅ 服务启动成功！${NC}"
        SERVER_IP=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "localhost")
        echo -e "${GREEN}🌐 Web面板: http://$SERVER_IP:5000${NC}"
    else
        echo -e "${RED}❌ 服务启动失败，请检查日志: journalctl -u cc-main -f${NC}"
        echo -e "${BLUE}尝试手动启动: systemctl start cc-main${NC}"
    fi
else
    echo -e "${GREEN}✅ 安装完成！${NC}"
    echo -e "${GREEN}🌐 启动命令: cd $PROJECT_DIR && ./start.sh${NC}"
    echo -e "${GREEN}🌐 Web面板: http://localhost:5000${NC}"
fi

echo
echo -e "${BLUE}📋 管理命令:${NC}"
if [ "$OS" = "linux" ]; then
    echo -e "  启动: systemctl start cc-main"
    echo -e "  停止: systemctl stop cc-main"
    echo -e "  状态: systemctl status cc-main"
    echo -e "  日志: journalctl -u cc-main -f"
else
    echo -e "  启动: cd $PROJECT_DIR && ./start.sh"
    echo -e "  停止: cd $PROJECT_DIR && ./stop.sh"
fi

echo
echo -e "${BLUE}⚙️  配置代理:${NC}"
echo -e "  SOCKS5: nano $PROJECT_DIR/config/socks5.txt"
echo -e "  HTTP: nano $PROJECT_DIR/config/http_proxies.txt"

echo
echo -e "${GREEN}🎉 安装完成！可以开始使用了！${NC}"