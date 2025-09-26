#!/bin/bash

# 安装测试脚本
# 用于验证安装是否成功

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

# 测试Python环境
test_python() {
    log_info "测试Python环境..."
    
    if command -v python3 &> /dev/null; then
        local version=$(python3 --version)
        log_success "Python版本: $version"
    else
        log_error "Python3 未安装"
        return 1
    fi
}

# 测试依赖
test_dependencies() {
    log_info "测试Python依赖..."
    
    if [ -f "verify_dependencies.py" ]; then
        python3 verify_dependencies.py
    else
        python3 -c "import flask, flask_socketio, psutil, socks" 2>/dev/null && {
            log_success "所有依赖已安装"
        } || {
            log_error "缺少必要依赖"
            return 1
        }
    fi
}

# 测试主程序
test_main_program() {
    log_info "测试主程序..."
    
    timeout 5 python3 main.py --help > /dev/null 2>&1 && {
        log_success "主程序正常"
    } || {
        log_error "主程序测试失败"
        return 1
    }
}

# 测试Web面板
test_web_panel() {
    log_info "测试Web面板..."
    
    timeout 5 python3 web_panel.py --help > /dev/null 2>&1 && {
        log_success "Web面板正常"
    } || {
        log_warning "Web面板测试失败，可能是依赖问题"
    }
}

# 测试配置文件
test_config_files() {
    log_info "测试配置文件..."
    
    local files=("socks5.txt" "http_proxies.txt" "accept_headers.txt" "referers.txt")
    
    for file in "${files[@]}"; do
        if [ -f "$file" ]; then
            log_success "配置文件存在: $file"
        else
            log_warning "配置文件缺失: $file"
        fi
    done
}

# 测试脚本权限
test_script_permissions() {
    log_info "测试脚本权限..."
    
    local scripts=("install.sh" "start_panel.sh" "uninstall.sh" "main.py" "web_panel.py")
    
    for script in "${scripts[@]}"; do
        if [ -x "$script" ]; then
            log_success "脚本可执行: $script"
        else
            log_warning "脚本不可执行: $script"
        fi
    done
}

# 显示测试结果
show_test_result() {
    echo
    echo "=================================="
    echo "测试完成"
    echo "=================================="
    echo
    echo "如果所有测试都通过，可以运行:"
    echo "  sudo ./install.sh"
    echo
    echo "或者手动启动:"
    echo "  ./start_panel.sh"
    echo
}

# 主函数
main() {
    echo "CC压测工具安装测试"
    echo "=================="
    echo
    
    test_python
    test_dependencies
    test_main_program
    test_web_panel
    test_config_files
    test_script_permissions
    
    show_test_result
}

# 运行测试
main "$@"
