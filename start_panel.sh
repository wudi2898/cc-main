#!/bin/bash

# CC压测工具Web控制面板启动脚本
# 支持多种启动模式和配置

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置
WEB_PORT=${WEB_PORT:-5000}
HOST=${HOST:-0.0.0.0}
DEBUG=${DEBUG:-false}
LOG_LEVEL=${LOG_LEVEL:-INFO}

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

# 显示帮助信息
show_help() {
    echo -e "${CYAN}CC压测工具Web控制面板启动脚本${NC}"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  -p, --port PORT     设置Web端口 (默认: 5000)"
    echo "  -h, --host HOST     设置监听地址 (默认: 0.0.0.0)"
    echo "  -d, --debug         启用调试模式"
    echo "  -b, --background    后台运行"
    echo "  -s, --status        查看运行状态"
    echo "  -k, --kill          停止运行中的服务"
    echo "  -r, --restart       重启服务"
    echo "  --help              显示此帮助信息"
    echo
    echo "示例:"
    echo "  $0                  # 默认启动"
    echo "  $0 -p 8080         # 在8080端口启动"
    echo "  $0 -d -b           # 调试模式后台运行"
    echo "  $0 -s              # 查看状态"
    echo "  $0 -k              # 停止服务"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    # 检查Python
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 未安装"
        exit 1
    fi
    
    # 检查必要模块
    python3 -c "import flask, flask_socketio, psutil" 2>/dev/null || {
        log_error "缺少必要依赖，请运行: pip install flask flask-socketio psutil"
        exit 1
    }
    
    log_success "依赖检查通过"
}

# 获取本机IP
get_local_ip() {
    if command -v hostname &> /dev/null; then
        hostname -I | awk '{print $1}' 2>/dev/null || echo "127.0.0.1"
    else
        echo "127.0.0.1"
    fi
}

# 检查端口是否被占用
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0  # 端口被占用
    else
        return 1  # 端口空闲
    fi
}

# 查找运行中的进程
find_running_process() {
    pgrep -f "web_panel.py" 2>/dev/null || echo ""
}

# 启动Web面板
start_web_panel() {
    local background=$1
    local debug=$2
    
    # 检查是否已经在运行
    local pid=$(find_running_process)
    if [ -n "$pid" ]; then
        log_warning "Web面板已在运行 (PID: $pid)"
        read -p "是否要重启？ [y/N]: " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kill_process $pid
        else
            return 0
        fi
    fi
    
    # 检查端口
    if check_port $WEB_PORT; then
        log_error "端口 $WEB_PORT 已被占用"
        log_info "请使用 -p 参数指定其他端口"
        exit 1
    fi
    
    # 创建日志目录
    mkdir -p logs
    
    # 获取本机IP
    local local_ip=$(get_local_ip)
    
    # 显示启动信息
    echo
    log_header "🚀 启动CC压测工具Web控制面板"
    echo
    echo -e "${CYAN}📡 本地访问:${NC} http://localhost:$WEB_PORT"
    echo -e "${CYAN}🌐 远程访问:${NC} http://$local_ip:$WEB_PORT"
    echo -e "${CYAN}🔧 监听地址:${NC} $HOST:$WEB_PORT"
    echo -e "${CYAN}🐛 调试模式:${NC} $debug"
    echo -e "${CYAN}📊 日志级别:${NC} $LOG_LEVEL"
    echo
    
    # 设置环境变量
    export FLASK_APP=web_panel.py
    export FLASK_ENV=development
    export PYTHONPATH=.
    
    if [ "$debug" = "true" ]; then
        export FLASK_DEBUG=1
    fi
    
    # 启动方式
    if [ "$background" = "true" ]; then
        log_info "后台启动Web面板..."
        nohup python3 web_panel.py > logs/web_panel.log 2>&1 &
        local new_pid=$!
        echo $new_pid > logs/web_panel.pid
        log_success "Web面板已在后台启动 (PID: $new_pid)"
        echo -e "${YELLOW}查看日志: tail -f logs/web_panel.log${NC}"
    else
        log_info "前台启动Web面板..."
        echo -e "${RED}按 Ctrl+C 停止服务${NC}"
        echo
        python3 web_panel.py
    fi
}

# 停止进程
kill_process() {
    local pid=$1
    if [ -n "$pid" ]; then
        log_info "停止进程 $pid..."
        kill -TERM $pid 2>/dev/null || true
        sleep 2
        if kill -0 $pid 2>/dev/null; then
            log_warning "强制停止进程 $pid..."
            kill -KILL $pid 2>/dev/null || true
        fi
        log_success "进程已停止"
    fi
}

# 停止服务
stop_service() {
    local pid=$(find_running_process)
    if [ -n "$pid" ]; then
        kill_process $pid
        # 清理PID文件
        rm -f logs/web_panel.pid
    else
        log_warning "没有找到运行中的Web面板进程"
    fi
}

# 查看状态
show_status() {
    local pid=$(find_running_process)
    local local_ip=$(get_local_ip)
    
    echo
    log_header "📊 CC压测工具状态"
    echo
    
    if [ -n "$pid" ]; then
        echo -e "${GREEN}状态:${NC} 运行中"
        echo -e "${GREEN}PID:${NC} $pid"
        echo -e "${GREEN}端口:${NC} $WEB_PORT"
        echo -e "${GREEN}本地访问:${NC} http://localhost:$WEB_PORT"
        echo -e "${GREEN}远程访问:${NC} http://$local_ip:$WEB_PORT"
        
        # 显示进程信息
        echo
        echo -e "${CYAN}进程信息:${NC}"
        ps -p $pid -o pid,ppid,cmd,etime,pcpu,pmem 2>/dev/null || true
        
        # 显示端口信息
        echo
        echo -e "${CYAN}端口信息:${NC}"
        lsof -i :$WEB_PORT 2>/dev/null || echo "无法获取端口信息"
        
    else
        echo -e "${RED}状态:${NC} 未运行"
        echo -e "${YELLOW}启动命令: $0${NC}"
    fi
    
    echo
}

# 重启服务
restart_service() {
    log_info "重启服务..."
    stop_service
    sleep 2
    start_web_panel "false" "false"
}

# 显示横幅
show_banner() {
    echo -e "${PURPLE}"
    echo "██████╗ ██████╗  ██████╗ ███████╗    ████████╗ ██████╗  ██████╗ ██╗     "
    echo "██╔══██╗██╔══██╗██╔═══██╗██╔════╝    ╚══██╔══╝██╔═══██╗██╔═══██╗██║     "
    echo "██║  ██║██║  ██║██║   ██║███████╗       ██║   ██║   ██║██║   ██║██║     "
    echo "██║  ██║██║  ██║██║   ██║╚════██║       ██║   ██║   ██║██║   ██║██║     "
    echo "██████╔╝██████╔╝╚██████╔╝███████║       ██║   ╚██████╔╝╚██████╔╝███████╗"
    echo "╚═════╝ ╚═════╝  ╚═════╝ ╚══════╝       ╚═╝    ╚═════╝  ╚═════╝ ╚══════╝"
    echo -e "${NC}"
    echo -e "${CYAN}          DDoS压力测试工具 - Web控制面板启动器${NC}"
    echo
}

# 主函数
main() {
    local background="false"
    local debug="false"
    local action="start"
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -p|--port)
                WEB_PORT="$2"
                shift 2
                ;;
            -h|--host)
                HOST="$2"
                shift 2
                ;;
            -d|--debug)
                debug="true"
                shift
                ;;
            -b|--background)
                background="true"
                shift
                ;;
            -s|--status)
                action="status"
                shift
                ;;
            -k|--kill)
                action="stop"
                shift
                ;;
            -r|--restart)
                action="restart"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 执行操作
    case $action in
        "start")
            show_banner
            check_dependencies
            start_web_panel "$background" "$debug"
            ;;
        "stop")
            stop_service
            ;;
        "restart")
            restart_service
            ;;
        "status")
            show_status
            ;;
    esac
}

# 错误处理
trap 'log_error "启动过程中出现错误！"; exit 1' ERR

# 运行主函数
main "$@"