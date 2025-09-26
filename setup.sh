#!/bin/bash

# DDoS压测工具一键安装脚本
# 支持自动依赖安装、配置检查、启动服务

set -e  # 出错时退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

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

# 检查操作系统
check_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
        DISTRO=$(lsb_release -si 2>/dev/null || echo "Unknown")
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        DISTRO="macOS"
    elif [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "msys" ]]; then
        OS="windows"
        DISTRO="Windows"
    else
        OS="unknown"
        DISTRO="Unknown"
    fi
    
    log_info "检测到操作系统: $DISTRO ($OS)"
}

# 检查Python版本
check_python() {
    log_info "检查Python环境..."
    
    if command -v python3 &> /dev/null; then
        PYTHON_VERSION=$(python3 --version | cut -d' ' -f2)
        PYTHON_MAJOR=$(echo $PYTHON_VERSION | cut -d'.' -f1)
        PYTHON_MINOR=$(echo $PYTHON_VERSION | cut -d'.' -f2)
        
        if [ "$PYTHON_MAJOR" -eq 3 ] && [ "$PYTHON_MINOR" -ge 7 ]; then
            log_success "Python版本: $PYTHON_VERSION ✓"
            PYTHON_CMD="python3"
        else
            log_error "Python版本过低: $PYTHON_VERSION (需要 >= 3.7)"
            exit 1
        fi
    else
        log_error "未找到Python3，请先安装Python 3.7+"
        exit 1
    fi
}

# 检查pip
check_pip() {
    log_info "检查pip..."
    
    if command -v pip3 &> /dev/null; then
        PIP_CMD="pip3"
    elif command -v pip &> /dev/null; then
        PIP_CMD="pip"
    else
        log_error "未找到pip，请先安装pip"
        exit 1
    fi
    
    log_success "发现pip: $PIP_CMD ✓"
}

# 安装Python依赖
install_dependencies() {
    log_info "安装Python依赖包..."
    
    # 检查requirements.txt是否存在
    if [ ! -f "requirements.txt" ]; then
        log_warning "requirements.txt 不存在，创建默认依赖文件"
        cat > requirements.txt << EOF
flask==2.3.3
flask-socketio==5.3.6
psutil==5.9.5
PySocks==1.7.1
EOF
    fi
    
    # 安装依赖
    log_info "正在安装依赖..."
    $PIP_CMD install -r requirements.txt
    
    # 验证关键依赖
    log_info "验证依赖安装..."
    $PYTHON_CMD -c "import flask, flask_socketio, psutil, socks" 2>/dev/null && {
        log_success "所有依赖安装成功 ✓"
    } || {
        log_error "依赖安装失败，请检查网络连接和权限"
        exit 1
    }
}

# 创建配置文件
create_config_files() {
    log_info "创建配置文件..."
    
    # 创建示例代理文件
    if [ ! -f "socks5.txt" ]; then
        log_info "创建示例SOCKS5代理文件..."
        cat > socks5.txt << EOF
127.0.0.1:1080
127.0.0.1:7890
EOF
        log_warning "请编辑 socks5.txt 添加真实的SOCKS5代理"
    fi
    
    # 创建HTTP代理文件
    if [ ! -f "http_proxies.txt" ]; then
        log_info "创建示例HTTP代理文件..."
        cat > http_proxies.txt << EOF
127.0.0.1:8080
127.0.0.1:3128
EOF
    fi
    
    # 检查headers配置文件
    if [ ! -f "accept_headers.txt" ]; then
        log_warning "accept_headers.txt 不存在，将使用内置默认配置"
    else
        HEADER_COUNT=$(wc -l < accept_headers.txt)
        log_success "加载了 $HEADER_COUNT 个Accept headers"
    fi
    
    # 检查referers配置文件
    if [ ! -f "referers.txt" ]; then
        log_warning "referers.txt 不存在，将使用内置默认配置"
    else
        REFERER_COUNT=$(wc -l < referers.txt)
        log_success "加载了 $REFERER_COUNT 个Referers"
    fi
}

# 设置文件权限
set_permissions() {
    log_info "设置文件权限..."
    
    # 设置脚本执行权限
    chmod +x *.py 2>/dev/null || true
    chmod +x *.sh 2>/dev/null || true
    
    # 创建日志目录
    mkdir -p logs
    
    log_success "文件权限设置完成 ✓"
}

# 系统优化（可选）
optimize_system() {
    if [ "$OS" = "linux" ]; then
        log_info "应用系统优化（需要root权限）..."
        
        if [ "$EUID" -eq 0 ]; then
            # 网络优化
            echo 'net.core.somaxconn = 65535' >> /etc/sysctl.conf
            echo 'net.ipv4.ip_local_port_range = 1024 65535' >> /etc/sysctl.conf
            echo 'net.core.netdev_max_backlog = 5000' >> /etc/sysctl.conf
            echo 'net.ipv4.tcp_fin_timeout = 30' >> /etc/sysctl.conf
            sysctl -p
            
            log_success "系统网络参数优化完成"
        else
            log_warning "非root用户，跳过系统优化"
            log_info "如需系统优化，请以root权限运行: sudo $0"
        fi
    else
        log_info "非Linux系统，跳过系统优化"
    fi
}

# 运行测试
run_tests() {
    log_info "运行基础测试..."
    
    # 测试主程序
    log_info "测试主程序..."
    timeout 5 $PYTHON_CMD main.py --help > /dev/null 2>&1 && {
        log_success "主程序测试通过 ✓"
    } || {
        log_error "主程序测试失败"
        return 1
    }
    
    # 测试Web面板
    log_info "测试Web面板..."
    timeout 5 $PYTHON_CMD web_panel.py --help > /dev/null 2>&1 && {
        log_success "Web面板测试通过 ✓"
    } || {
        log_warning "Web面板测试失败，可能是依赖问题"
    }
    
    # 测试代理连接（如果有代理的话）
    if [ -f "socks5.txt" ] && [ -s "socks5.txt" ]; then
        log_info "测试代理连接..."
        timeout 10 $PYTHON_CMD main.py check https://httpbin.org/ip 1 1 > /dev/null 2>&1 && {
            log_success "代理连接测试通过 ✓"
        } || {
            log_warning "代理连接测试失败，请检查代理配置"
        }
    fi
}

# 启动服务
start_services() {
    log_info "准备启动服务..."
    
    # 获取本机IP
    if [ "$OS" = "macos" ]; then
        LOCAL_IP=$(ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1)
    else
        LOCAL_IP=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "127.0.0.1")
    fi
    
    echo
    log_header "🚀 安装完成！服务启动指南"
    echo
    echo -e "${CYAN}1. 启动Web控制面板:${NC}"
    echo -e "   ${GREEN}python3 web_panel.py${NC}"
    echo -e "   ${YELLOW}访问地址: http://localhost:5000${NC}"
    echo -e "   ${YELLOW}远程访问: http://$LOCAL_IP:5000${NC}"
    echo
    echo -e "${CYAN}2. 命令行使用示例:${NC}"
    echo -e "   ${GREEN}# 基础攻击${NC}"
    echo -e "   python3 main.py cc https://target.com 100 10"
    echo
    echo -e "   ${GREEN}# 超负荷模式${NC}"
    echo -e "   python3 main.py cc https://target.com 500 50 --overload --fire-and-forget"
    echo
    echo -e "   ${GREEN}# CF绕过模式${NC}"
    echo -e "   python3 main.py cc https://cf-site.com 100 10 --cf-bypass"
    echo
    echo -e "${CYAN}3. 性能测试:${NC}"
    echo -e "   ${GREEN}python3 performance_test.py${NC}"
    echo -e "   ${GREEN}python3 proxy_benchmark.py${NC}"
    echo
    echo -e "${CYAN}4. 配置文件:${NC}"
    echo -e "   ${YELLOW}代理列表: socks5.txt, http_proxies.txt${NC}"
    echo -e "   ${YELLOW}请求头: accept_headers.txt${NC}"
    echo -e "   ${YELLOW}引用页: referers.txt${NC}"
    echo
    
    # 询问是否立即启动Web面板
    read -p "$(echo -e ${CYAN}是否立即启动Web控制面板？ [y/N]: ${NC})" -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "启动Web控制面板..."
        echo -e "${GREEN}Web面板启动中...${NC}"
        echo -e "${YELLOW}访问地址: http://localhost:5000${NC}"
        echo -e "${YELLOW}远程访问: http://$LOCAL_IP:5000${NC}"
        echo -e "${RED}按 Ctrl+C 停止服务${NC}"
        echo
        $PYTHON_CMD web_panel.py
    else
        echo -e "${YELLOW}稍后可手动启动: python3 web_panel.py${NC}"
    fi
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
    echo -e "${CYAN}          DDoS压力测试工具 - 超负荷增强版 v3.0.0${NC}"
    echo -e "${YELLOW}              一键安装脚本 - 自动配置所有依赖${NC}"
    echo
}

# 主函数
main() {
    show_banner
    
    log_header "开始安装DDoS压测工具"
    
    # 检查环境
    check_os
    check_python
    check_pip
    
    # 安装和配置
    install_dependencies
    create_config_files
    set_permissions
    
    # 可选优化
    read -p "$(echo -e ${CYAN}是否进行系统网络优化？ [y/N]: ${NC})" -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        optimize_system
    fi
    
    # 测试
    run_tests
    
    # 启动服务
    start_services
}

# 错误处理
trap 'log_error "安装过程中出现错误！"; exit 1' ERR

# 运行主函数
main "$@"
