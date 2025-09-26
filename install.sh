#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 项目信息
PROJECT_NAME="CC压力测试工具"
VERSION="2.0.0"

# 默认配置
DEFAULT_PORT="8080"
DEFAULT_TASKS_FILE="/cc-tasks.json"

# 解析命令行参数
PORT=$DEFAULT_PORT
TASKS_FILE=$DEFAULT_TASKS_FILE

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -t|--tasks-file)
            TASKS_FILE="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  -p, --port PORT        设置服务器端口 (默认: $DEFAULT_PORT)"
            echo "  -t, --tasks-file FILE  设置任务文件路径 (默认: $DEFAULT_TASKS_FILE)"
            echo "  -h, --help             显示帮助信息"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 -h 或 --help 查看帮助"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    ${PROJECT_NAME} v${VERSION}                    ║${NC}"
echo -e "${BLUE}║                        一键安装运行                        ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# 检测操作系统
OS=$(uname -s)
ARCH=$(uname -m)

echo -e "${CYAN}🔍 检测系统环境...${NC}"
echo -e "${GREEN}✅ 操作系统: $OS${NC}"
echo -e "${GREEN}✅ 架构: $ARCH${NC}"

# 停止相关进程
echo -e "${CYAN}🛑 停止相关进程...${NC}"
pkill -f "cc-go" 2>/dev/null || true
pkill -f "api_server" 2>/dev/null || true
pkill -f "main.go" 2>/dev/null || true
sleep 2
echo -e "${GREEN}✅ 相关进程已停止${NC}"

# 检查并安装Go
echo -e "${CYAN}📦 检查Go环境...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}⚠️  Go未安装，开始自动安装...${NC}"
    
    if [[ "$OS" == "Darwin" ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            brew install go
        else
            echo -e "${YELLOW}⚠️  未找到Homebrew，请手动安装Go${NC}"
            echo -e "${BLUE}📥 下载地址: https://golang.org/dl/${NC}"
            exit 1
        fi
    elif [[ "$OS" == "Linux" ]]; then
        # Linux
        if command -v apt-get &> /dev/null; then
            sudo apt-get update
            sudo apt-get install -y golang-go
        elif command -v yum &> /dev/null; then
            sudo yum install -y golang
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y golang
        else
            echo -e "${YELLOW}⚠️  未找到包管理器，请手动安装Go${NC}"
            echo -e "${BLUE}📥 下载地址: https://golang.org/dl/${NC}"
            exit 1
        fi
    else
        echo -e "${RED}❌ 不支持的操作系统: $OS${NC}"
        exit 1
    fi
else
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✅ Go已安装，版本: $GO_VERSION${NC}"
fi

# 设置Go环境变量
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# 检查Go版本
GO_VERSION_NUM=$(go version | awk '{print $3}' | sed 's/go//' | cut -d. -f2)
if [ "$GO_VERSION_NUM" -lt 21 ]; then
    echo -e "${RED}❌ Go版本过低，需要1.21+，当前版本: $(go version)${NC}"
    exit 1
fi

# 安装依赖
echo -e "${CYAN}📦 安装依赖...${NC}"
go mod tidy
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 依赖安装失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 依赖安装完成${NC}"

# 构建程序
echo -e "${CYAN}🔨 构建程序...${NC}"

# 构建主程序
echo -e "${BLUE}📦 构建主程序...${NC}"
go build -ldflags="-s -w" -o cc-go main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 主程序构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 主程序构建完成${NC}"

# 构建API服务器
echo -e "${BLUE}📦 构建API服务器...${NC}"
go build -ldflags="-s -w" -o api_server api_server.go
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ API服务器构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ API服务器构建完成${NC}"

# 设置权限
chmod +x cc-go api_server

# 创建任务存储文件
echo "[]" > /cc-tasks.json
chmod 666 /cc-tasks.json

# 检查前端文件
echo -e "${CYAN}🎨 检查前端文件...${NC}"
if [ ! -d "frontend" ]; then
    echo -e "${RED}❌ 前端目录不存在${NC}"
    exit 1
fi

if [ ! -f "frontend/css/bootstrap.min.css" ]; then
    echo -e "${YELLOW}⚠️  下载Bootstrap CSS...${NC}"
    mkdir -p frontend/css
    curl -s -o frontend/css/bootstrap.min.css https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css
fi

if [ ! -f "frontend/js/bootstrap.bundle.min.js" ]; then
    echo -e "${YELLOW}⚠️  下载Bootstrap JS...${NC}"
    mkdir -p frontend/js
    curl -s -o frontend/js/bootstrap.bundle.min.js https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js
fi

echo -e "${GREEN}✅ 前端文件检查完成${NC}"

# 创建系统服务（可选）
echo -e "${CYAN}🔧 创建系统服务...${NC}"
if [[ "$OS" == "Linux" ]]; then
    # 创建systemd服务
    sudo tee /etc/systemd/system/cc-main.service > /dev/null <<EOF
[Unit]
Description=CC压力测试工具
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$(pwd)
ExecStart=$(pwd)/api_server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    echo -e "${GREEN}✅ 系统服务创建完成${NC}"
    echo -e "${BLUE}💡 使用 'sudo systemctl start cc-main' 启动服务${NC}"
    echo -e "${BLUE}💡 使用 'sudo systemctl enable cc-main' 设置开机自启${NC}"
fi

# 启动服务
echo -e "${GREEN}🚀 启动服务...${NC}"
echo -e "${YELLOW}📱 前端地址: http://localhost:8080${NC}"
echo -e "${YELLOW}🔗 API地址: http://localhost:8080/api${NC}"
echo -e "${YELLOW}📊 日志页面: http://localhost:8080/logs.html${NC}"
echo -e "${YELLOW}🛡️  CF绕过: 已启用${NC}"
echo -e "${YELLOW}🎯 随机化: 上亿万万个组合${NC}"
echo ""
echo -e "${BLUE}按 Ctrl+C 停止服务${NC}"
echo ""

# 启动API服务器
./api_server
