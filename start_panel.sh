#!/bin/bash

# DDoS压测工具 Web控制面板启动脚本
# 自动检查依赖、配置并启动服务

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 显示Logo
show_logo() {
    clear
    echo -e "${PURPLE}"
    echo "██╗    ██╗███████╗██████╗     ██████╗  █████╗ ███╗   ██╗███████╗██╗     "
    echo "██║    ██║██╔════╝██╔══██╗    ██╔══██╗██╔══██╗████╗  ██║██╔════╝██║     "
    echo "██║ █╗ ██║█████╗  ██████╔╝    ██████╔╝███████║██╔██╗ ██║█████╗  ██║     "
    echo "██║███╗██║██╔══╝  ██╔══██╗    ██╔═══╝ ██╔══██║██║╚██╗██║██╔══╝  ██║     "
    echo "╚███╔███╔╝███████╗██████╔╝    ██║     ██║  ██║██║ ╚████║███████╗███████╗"
    echo " ╚══╝╚══╝ ╚══════╝╚═════╝     ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝"
    echo -e "${NC}"
    echo -e "${CYAN}           DDoS压测工具 Web控制面板 v3.0.0${NC}"
    echo
}

# 检查Python环境
check_python() {
    if ! command -v python3 &> /dev/null; then
        echo -e "${RED}❌ 错误: 未找到Python3${NC}"
        echo "请先安装Python 3.7+: https://python.org"
        exit 1
    fi
    
    PYTHON_VERSION=$(python3 --version | cut -d' ' -f2)
    echo -e "${GREEN}✅ Python版本: $PYTHON_VERSION${NC}"
}

# 检查依赖
check_dependencies() {
    echo -e "${BLUE}🔍 检查依赖...${NC}"
    
    # 检查核心依赖
    python3 -c "import flask, flask_socketio, psutil" 2>/dev/null || {
        echo -e "${YELLOW}⚠️  缺少依赖，正在安装...${NC}"
        pip3 install flask flask-socketio psutil PySocks
    }
    
    echo -e "${GREEN}✅ 依赖检查完成${NC}"
}

# 检查配置文件
check_config() {
    echo -e "${BLUE}📁 检查配置文件...${NC}"
    
    # 检查代理文件
    if [ ! -f "socks5.txt" ] || [ ! -s "socks5.txt" ]; then
        echo -e "${YELLOW}⚠️  代理文件不存在，创建示例文件${NC}"
        echo "127.0.0.1:1080" > socks5.txt
        echo "127.0.0.1:7890" >> socks5.txt
        echo -e "${CYAN}💡 请编辑 socks5.txt 添加真实代理${NC}"
    fi
    
    # 检查其他配置文件
    [ -f "accept_headers.txt" ] && echo -e "${GREEN}✅ Headers配置: $(wc -l < accept_headers.txt) 条${NC}"
    [ -f "referers.txt" ] && echo -e "${GREEN}✅ Referer配置: $(wc -l < referers.txt) 条${NC}"
    
    echo -e "${GREEN}✅ 配置检查完成${NC}"
}

# 启动Web面板
start_panel() {
    echo -e "${BLUE}🚀 启动Web控制面板...${NC}"
    echo
    
    # 检查端口占用
    if lsof -Pi :5000 -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${RED}❌ 端口5000已被占用${NC}"
        echo "请关闭占用端口的程序或修改web_panel.py中的端口号"
        exit 1
    fi
    
    # 启动面板
    python3 web_panel.py
}

# 主函数
main() {
    show_logo
    
    echo -e "${CYAN}🔧 准备启动Web控制面板...${NC}"
    echo
    
    # 检查环境
    check_python
    check_dependencies
    check_config
    
    echo
    echo -e "${GREEN}🎉 环境检查完成，正在启动...${NC}"
    echo
    
    # 启动面板
    start_panel
}

# 错误处理
trap 'echo -e "\n${RED}❌ 启动失败${NC}"; exit 1' ERR

# 运行
main "$@"
