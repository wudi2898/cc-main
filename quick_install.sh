#!/bin/bash

# 快速安装脚本 - 仅安装核心依赖
# 用于快速测试和开发环境

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

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Python
check_python() {
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 未安装"
        exit 1
    fi
    
    local version=$(python3 --version | cut -d' ' -f2)
    log_success "Python版本: $version"
}

# 安装最小依赖
install_minimal() {
    log_info "安装最小依赖..."
    
    if [ -f "requirements-minimal.txt" ]; then
        pip3 install -r requirements-minimal.txt
    else
        log_info "安装核心依赖包..."
        pip3 install flask==2.3.3 flask-socketio==5.3.6 psutil==5.9.5 PySocks==1.7.1
    fi
    
    log_success "最小依赖安装完成"
}

# 验证安装
verify_install() {
    log_info "验证安装..."
    
    if [ -f "verify_dependencies.py" ]; then
        python3 verify_dependencies.py
    else
        python3 -c "import flask, flask_socketio, psutil, socks" && {
            log_success "依赖验证通过"
        } || {
            log_error "依赖验证失败"
            exit 1
        }
    fi
}

# 创建必要目录
create_dirs() {
    log_info "创建必要目录..."
    
    mkdir -p config
    mkdir -p logs
    mkdir -p templates
    
    log_success "目录创建完成"
}

# 创建示例配置
create_config() {
    log_info "创建示例配置..."
    
    if [ ! -f "config/socks5.txt" ]; then
        cat > config/socks5.txt << EOF
# SOCKS5代理列表
# 格式: IP:端口
# 示例:
# 127.0.0.1:1080
EOF
        log_warning "请编辑 config/socks5.txt 添加真实代理"
    fi
    
    log_success "配置创建完成"
}

# 显示使用说明
show_usage() {
    echo
    echo "🎉 快速安装完成！"
    echo
    echo "📋 下一步操作："
    echo "1. 编辑代理配置: nano config/socks5.txt"
    echo "2. 启动Web面板: python3 web_panel.py"
    echo "3. 访问控制面板: http://localhost:5000"
    echo
    echo "📚 更多信息："
    echo "- 完整安装: ./install.sh"
    echo "- 依赖验证: python3 verify_dependencies.py"
    echo "- 使用说明: cat README.md"
    echo
}

# 主函数
main() {
    echo "🚀 CC压测工具快速安装"
    echo "===================="
    echo
    
    check_python
    install_minimal
    create_dirs
    create_config
    verify_install
    show_usage
}

# 运行主函数
main "$@"
