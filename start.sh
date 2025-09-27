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
AUTHOR="优化版"

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    ${PROJECT_NAME} v${VERSION}                    ║${NC}"
echo -e "${BLUE}║                        ${AUTHOR}                        ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# 系统检查
echo -e "${CYAN}🔍 系统环境检查...${NC}"

# 检查操作系统
OS=$(uname -s)
if [[ "$OS" == "Linux" ]]; then
    echo -e "${GREEN}✅ 检测到Linux系统${NC}"
elif [[ "$OS" == "Darwin" ]]; then
    echo -e "${GREEN}✅ 检测到macOS系统${NC}"
else
    echo -e "${YELLOW}⚠️  未识别的操作系统: $OS${NC}"
fi

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go未安装，请先安装Go 1.21+${NC}"
    echo -e "${YELLOW}📥 下载地址: https://golang.org/dl/${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✅ Go版本: $GO_VERSION${NC}"

# 检查网络连接
if ping -c 1 8.8.8.8 &> /dev/null; then
    echo -e "${GREEN}✅ 网络连接正常${NC}"
else
    echo -e "${YELLOW}⚠️  网络连接异常，可能影响代理功能${NC}"
fi

echo ""

# 依赖管理
echo -e "${CYAN}📦 依赖管理...${NC}"
go mod tidy
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 依赖管理失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 依赖检查完成${NC}"

# 代码检查
echo -e "${CYAN}🔍 代码质量检查...${NC}"
go vet ./...
if [ $? -ne 0 ]; then
    echo -e "${YELLOW}⚠️  代码检查发现问题，但继续构建${NC}"
fi

# 构建优化
echo -e "${CYAN}🔨 构建优化...${NC}"

# 构建主程序
echo -e "${BLUE}📦 构建主程序...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$VERSION" -o cc-go main.go
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 主程序构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 主程序构建完成${NC}"

# 构建API服务器
echo -e "${BLUE}📦 构建API服务器...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o api_server api_server.go
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ API服务器构建失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ API服务器构建完成${NC}"

# 文件权限设置
chmod +x cc-go api_server


# 性能优化建议
echo -e "${CYAN}⚡ 性能优化建议...${NC}"
echo -e "${PURPLE}💡 建议配置:${NC}"
echo -e "   - 线程数: 10000-50000"
echo -e "   - RPS: 50000-200000"
echo -e "   - 超时: 10-30秒"
echo -e "   - 使用SOCKS5代理池"
echo ""

# 启动服务
echo -e "${GREEN}🚀 启动服务...${NC}"
echo -e "${YELLOW}📱 前端地址: http://localhost:8080${NC}"
echo -e "${YELLOW}🔗 API地址: http://localhost:8080/api${NC}"
echo -e "${YELLOW}📊 日志页面: http://localhost:8080/logs.html${NC}"
echo -e "${YELLOW}🛡️  CF绕过: 已启用${NC}"
echo ""
echo -e "${BLUE}按 Ctrl+C 停止服务${NC}"
echo ""

# 启动API服务器
./api_server
