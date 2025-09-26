#!/bin/bash

# CC压测工具 - 真正的一键脚本
# 整合：下载 + 安装 + 性能优化 + 配置 + 启动

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

log_header() {
    echo -e "${PURPLE}================================${NC}"
    echo -e "${PURPLE}$1${NC}"
    echo -e "${PURPLE}================================${NC}"
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
    echo -e "${CYAN}          DDoS压力测试工具 - 真正的一键脚本${NC}"
    echo -e "${YELLOW}              下载+安装+优化+配置+启动 - 一条命令搞定${NC}"
    echo
}

# 检查root权限
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "此脚本需要root权限运行"
        log_info "请使用: sudo $0"
        exit 1
    fi
}

# 检测系统
detect_system() {
    log_info "检测系统信息..."
    
    # 检测操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    else
        OS=$(uname -s)
        VER=$(uname -r)
    fi
    
    # 检测CPU和内存
    CPU_CORES=$(nproc)
    MEMORY_GB=$(free -g | grep "Mem:" | awk '{print $2}')
    
    log_success "系统: $OS $VER"
    log_success "CPU: $CPU_CORES 核心"
    log_success "内存: ${MEMORY_GB}GB"
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

# 下载项目
download_project() {
    log_info "下载项目..."
    
    # 如果已存在，先删除
    if [ -d "cc-main" ]; then
        rm -rf cc-main
    fi
    
    # 下载项目
    git clone https://github.com/wudi2898/cc-main.git
    cd cc-main
    
    log_success "项目下载完成"
}

# 安装项目
install_project() {
    log_info "安装项目..."
    
    # 直接运行安装脚本
    chmod +x install.sh
    ./install.sh
    
    log_success "项目安装完成"
}

# 优化内核参数
optimize_kernel() {
    log_info "优化内核参数..."
    
    # 备份原始配置
    cp /etc/sysctl.conf /etc/sysctl.conf.backup.$(date +%Y%m%d_%H%M%S)
    
    # 网络优化
    cat >> /etc/sysctl.conf << EOF

# CC压测工具网络优化 - $(date)
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 5000
net.core.rmem_default = 262144
net.core.rmem_max = 16777216
net.core.wmem_default = 262144
net.core.wmem_max = 16777216
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_keepalive_time = 1200
net.ipv4.tcp_keepalive_intvl = 15
net.ipv4.tcp_keepalive_probes = 5
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_congestion_control = bbr
net.ipv4.ip_local_port_range = 1024 65535
net.netfilter.nf_conntrack_max = 1048576
net.netfilter.nf_conntrack_tcp_timeout_established = 1200
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
fs.file-max = 2097152
fs.nr_open = 1048576
kernel.pid_max = 4194304
kernel.threads-max = 2097152
EOF
    
    # 应用配置
    sysctl -p
    
    log_success "内核参数优化完成"
}

# 优化文件描述符限制
optimize_limits() {
    log_info "优化文件描述符限制..."
    
    # 备份原始配置
    cp /etc/security/limits.conf /etc/security/limits.conf.backup.$(date +%Y%m%d_%H%M%S)
    
    # 添加限制配置
    cat >> /etc/security/limits.conf << EOF

# CC压测工具文件描述符限制 - $(date)
* soft nofile 1048576
* hard nofile 1048576
* soft nproc 1048576
* hard nproc 1048576
root soft nofile 1048576
root hard nofile 1048576
root soft nproc 1048576
root hard nproc 1048576
EOF
    
    # 配置systemd限制
    mkdir -p /etc/systemd/system.conf.d
    cat > /etc/systemd/system.conf.d/limits.conf << EOF
[Manager]
DefaultLimitNOFILE=1048576
DefaultLimitNPROC=1048576
EOF
    
    log_success "文件描述符限制优化完成"
}

# 优化网络接口
optimize_network() {
    log_info "优化网络接口..."
    
    # 获取网络接口
    INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    
    if [ -n "$INTERFACE" ]; then
        # 优化网卡队列
        echo 0 > /proc/sys/net/core/netdev_budget
        echo 600 > /proc/sys/net/core/netdev_budget
        
        # 设置网卡参数
        if [ -f "/sys/class/net/$INTERFACE/queues/rx-0/rps_cpus" ]; then
            echo f > /sys/class/net/$INTERFACE/queues/rx-0/rps_cpus
        fi
        
        log_success "网络接口 $INTERFACE 优化完成"
    fi
}

# 优化TCP拥塞控制
optimize_tcp() {
    log_info "优化TCP拥塞控制..."
    
    # 检查BBR支持
    if modprobe tcp_bbr 2>/dev/null; then
        echo 'tcp_bbr' >> /etc/modules-load.d/tcp_bbr.conf
        echo 'net.core.default_qdisc=fq' >> /etc/sysctl.d/99-tcp-optimization.conf
        echo 'net.ipv4.tcp_congestion_control=bbr' >> /etc/sysctl.d/99-tcp-optimization.conf
        log_success "BBR拥塞控制已启用"
    else
        log_warning "BBR不支持，使用默认拥塞控制"
    fi
}

# 优化内存管理
optimize_memory() {
    log_info "优化内存管理..."
    
    # 禁用swap（如果内存足够）
    if [ "$MEMORY_GB" -gt 4 ]; then
        swapoff -a
        sed -i '/swap/d' /etc/fstab
        log_success "已禁用swap（内存: ${MEMORY_GB}GB）"
    else
        log_warning "内存较少(${MEMORY_GB}GB)，保留swap"
    fi
    
    # 优化内存回收
    echo 1 > /proc/sys/vm/drop_caches
    echo 3 > /proc/sys/vm/drop_caches
    
    log_success "内存管理优化完成"
}

# 优化进程调度
optimize_scheduler() {
    log_info "优化进程调度..."
    
    # 设置CPU调度策略
    echo performance > /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor 2>/dev/null || true
    
    # 优化进程调度参数
    echo 1 > /proc/sys/kernel/sched_rt_runtime_us
    echo 950000 > /proc/sys/kernel/sched_rt_period_us
    
    log_success "进程调度优化完成"
}

# 优化系统服务
optimize_services() {
    log_info "优化系统服务..."
    
    # 禁用不必要的服务
    SERVICES_TO_DISABLE=(
        "bluetooth"
        "cups"
        "avahi-daemon"
        "modemmanager"
        "snapd"
        "snapd.socket"
    )
    
    for service in "${SERVICES_TO_DISABLE[@]}"; do
        if systemctl is-enabled "$service" &>/dev/null; then
            systemctl disable "$service" &>/dev/null || true
            systemctl stop "$service" &>/dev/null || true
        fi
    done
    
    # 优化systemd配置
    cat > /etc/systemd/system.conf.d/optimization.conf << EOF
[Manager]
DefaultTimeoutStartSec=10s
DefaultTimeoutStopSec=10s
DefaultRestartSec=100ms
EOF
    
    systemctl daemon-reload
    
    log_success "系统服务优化完成"
}

# 优化磁盘I/O
optimize_disk() {
    log_info "优化磁盘I/O..."
    
    # 优化磁盘调度器
    for disk in /sys/block/sd*; do
        if [ -d "$disk" ]; then
            echo mq-deadline > "$disk/queue/scheduler" 2>/dev/null || true
        fi
    done
    
    # 优化磁盘参数
    echo 0 > /proc/sys/vm/swappiness
    echo 1 > /proc/sys/vm/overcommit_memory
    
    log_success "磁盘I/O优化完成"
}

# 配置防火墙
configure_firewall() {
    log_info "配置防火墙..."
    
    # 尝试开放5000端口
    if command -v ufw &> /dev/null; then
        ufw allow 5000
        log_success "UFW防火墙已配置"
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-port=5000/tcp
        firewall-cmd --reload
        log_success "Firewalld防火墙已配置"
    else
        # 使用iptables
        iptables -A INPUT -p tcp --dport 5000 -j ACCEPT
        log_success "iptables防火墙已配置"
    fi
}

# 配置代理
configure_proxy() {
    log_info "配置代理..."
    
    # 创建代理配置
    cat > /opt/cc-main/config/socks5.txt << EOF
# SOCKS5代理列表 - 请添加真实代理
# 格式: IP:端口
# 示例:
# 127.0.0.1:1080
# 192.168.1.100:7890
# proxy.example.com:1080

# 临时测试代理（请替换为真实代理）
127.0.0.1:1080
EOF
    
    log_warning "请编辑 /opt/cc-main/config/socks5.txt 添加真实代理"
}

# 创建性能监控脚本
create_monitoring() {
    log_info "创建性能监控脚本..."
    
    cat > /usr/local/bin/cc-monitor << 'EOF'
#!/bin/bash
# CC压测工具性能监控脚本

echo "=== CC压测工具性能监控 ==="
echo "时间: $(date)"
echo

echo "=== 系统负载 ==="
uptime
echo

echo "=== CPU使用率 ==="
top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1
echo

echo "=== 内存使用 ==="
free -h
echo

echo "=== 网络连接 ==="
ss -tuln | wc -l
echo

echo "=== 文件描述符 ==="
lsof | wc -l
echo

echo "=== 进程数 ==="
ps aux | wc -l
echo

echo "=== 网络流量 ==="
cat /proc/net/dev | grep -v "lo:" | awk '{print $1 " RX:" $2 " TX:" $10}'
EOF
    
    chmod +x /usr/local/bin/cc-monitor
    
    log_success "性能监控脚本创建完成"
}

# 启动服务
start_service() {
    log_info "启动服务..."
    
    # 直接使用systemctl启动服务
    systemctl start cc-main
    
    # 等待服务启动
    sleep 5
    
    # 检查服务状态
    if systemctl is-active --quiet cc-main; then
        log_success "服务启动成功"
        log_info "Web控制面板地址: http://$(hostname -I | awk '{print $1}'):5000"
    else
        log_warning "服务启动可能有问题，请检查日志: journalctl -u cc-main -f"
    fi
}

# 显示最终结果
show_result() {
    local server_ip=$(hostname -I | awk '{print $1}')
    
    echo
    log_header "🎉 一键安装+优化完成！"
    echo
    echo -e "${GREEN}✅ 已完成的操作:${NC}"
    echo "  • 系统依赖安装"
    echo "  • CC压测工具安装"
    echo "  • 服务器性能优化"
    echo "  • 系统配置优化"
    echo "  • 防火墙配置"
    echo "  • 代理配置"
    echo "  • 服务自动启动"
    echo
    echo -e "${CYAN}📋 访问信息:${NC}"
    echo -e "  ${YELLOW}Web面板:${NC} http://localhost:5000"
    echo -e "  ${YELLOW}远程访问:${NC} http://$server_ip:5000"
    echo
    echo -e "${CYAN}📊 性能监控:${NC}"
    echo -e "  ${GREEN}cc-monitor${NC} - 查看系统性能"
    echo
    echo -e "${CYAN}🔧 管理命令:${NC}"
    echo -e "  ${GREEN}systemctl start cc-main${NC}    - 启动服务"
    echo -e "  ${GREEN}systemctl stop cc-main${NC}     - 停止服务"
    echo -e "  ${GREEN}systemctl restart cc-main${NC}  - 重启服务"
    echo -e "  ${GREEN}systemctl status cc-main${NC}   - 查看状态"
    echo -e "  ${GREEN}journalctl -u cc-main -f${NC}   - 查看日志"
    echo
    echo -e "${CYAN}⚙️  配置代理:${NC}"
    echo -e "  ${YELLOW}编辑代理:${NC} nano /opt/cc-main/config/socks5.txt"
    echo -e "  ${YELLOW}重启服务:${NC} systemctl restart cc-main"
    echo
    echo -e "${RED}⚠️  重要提醒:${NC}"
    echo -e "  • 请编辑代理配置文件添加真实代理"
    echo -e "  • 建议重启服务器以确保所有优化生效"
    echo -e "  • 仅用于授权的安全测试"
    echo -e "  • 遵守当地法律法规"
    echo
    echo -e "${GREEN}🚀 服务器已优化至最佳性能状态，可以开始使用了！${NC}"
    echo
}

# 主函数
main() {
    show_banner
    
    log_info "开始一键安装+优化..."
    
    # 检查权限
    check_root
    
    # 检测系统
    detect_system
    
    # 安装系统依赖
    install_system_deps
    
    # 下载项目
    download_project
    
    # 安装项目
    install_project
    
    # 性能优化
    optimize_kernel
    optimize_limits
    optimize_network
    optimize_tcp
    optimize_memory
    optimize_scheduler
    optimize_services
    optimize_disk
    
    # 配置
    configure_firewall
    configure_proxy
    create_monitoring
    
    # 启动服务
    start_service
    
    # 显示结果
    show_result
}

# 错误处理
trap 'log_error "安装过程中出现错误！"; exit 1' ERR

# 运行主函数
main "$@"
