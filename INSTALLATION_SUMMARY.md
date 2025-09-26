# 📋 安装优化总结

## 🎯 优化内容

### 1. 一键安装脚本 (`install.sh`)
- ✅ **自动环境检测**: 操作系统、Python版本、依赖检查
- ✅ **系统服务创建**: systemd/launchd服务配置
- ✅ **用户权限管理**: 创建专用用户和目录
- ✅ **开机自启配置**: 自动设置系统启动
- ✅ **系统优化**: 网络参数和文件描述符优化
- ✅ **管理命令**: 创建便捷的管理脚本

### 2. 启动管理脚本 (`start_panel.sh`)
- ✅ **多种启动模式**: 前台、后台、调试模式
- ✅ **端口管理**: 自动检测端口占用
- ✅ **进程管理**: 启动、停止、重启、状态查看
- ✅ **日志管理**: 实时日志查看和导出
- ✅ **参数配置**: 支持自定义端口和主机

### 3. 卸载脚本 (`uninstall.sh`)
- ✅ **完全清理**: 删除所有安装的文件和配置
- ✅ **服务清理**: 停止并删除系统服务
- ✅ **用户清理**: 删除专用用户和目录
- ✅ **配置恢复**: 恢复系统配置文件
- ✅ **安全确认**: 防止误删的确认机制

### 4. 系统服务配置
- ✅ **Linux支持**: systemd服务配置
- ✅ **macOS支持**: launchd服务配置
- ✅ **自动重启**: 服务异常时自动重启
- ✅ **日志管理**: 系统级日志记录
- ✅ **安全设置**: 最小权限原则

### 5. 管理命令
- ✅ **cc-start**: 启动服务
- ✅ **cc-stop**: 停止服务
- ✅ **cc-restart**: 重启服务
- ✅ **cc-status**: 查看状态
- ✅ **cc-logs**: 查看日志

## 🚀 使用方法

### 一键安装
```bash
git clone https://github.com/wudi2898/cc-main.git
cd cc-main
sudo ./install.sh
```

### 服务管理
```bash
cc-start      # 启动
cc-stop       # 停止
cc-restart    # 重启
cc-status     # 状态
cc-logs       # 日志
```

### 手动启动
```bash
./start_panel.sh          # 前台启动
./start_panel.sh -b       # 后台启动
./start_panel.sh -d       # 调试模式
./start_panel.sh -p 8080  # 指定端口
```

### 完全卸载
```bash
sudo ./uninstall.sh
```

## 📁 文件结构

```
cc-main/
├── install.sh              # 一键安装脚本
├── start_panel.sh          # 启动管理脚本
├── uninstall.sh            # 卸载脚本
├── test_install.sh         # 安装测试脚本
├── main.py                 # 主程序
├── web_panel.py            # Web控制面板
├── requirements.txt        # Python依赖
├── README.md               # 项目说明
├── QUICKSTART.md           # 快速启动指南
├── INSTALLATION_SUMMARY.md # 安装总结
└── templates/
    └── index.html          # Web界面模板
```

## 🔧 安装目录结构

```
/opt/cc-main/
├── main.py                 # 主程序
├── web_panel.py            # Web面板
├── venv/                   # Python虚拟环境
├── logs/                   # 日志目录
├── config/                 # 配置文件
├── socks5.txt              # SOCKS5代理
├── http_proxies.txt        # HTTP代理
└── templates/              # Web模板
```

## 🌐 访问地址

- **本地访问**: http://localhost:5000
- **远程访问**: http://你的IP:5000

## ⚠️ 注意事项

1. **权限要求**: 安装脚本需要root权限
2. **系统支持**: 支持Linux和macOS
3. **Python版本**: 需要Python 3.7+
4. **代理配置**: 需要配置有效的代理列表
5. **合法使用**: 仅用于授权的安全测试

## 🛠️ 故障排除

### 安装失败
```bash
# 检查权限
sudo ./install.sh

# 检查依赖
./test_install.sh
```

### 服务启动失败
```bash
# 查看状态
cc-status

# 查看日志
cc-logs

# 重启服务
cc-restart
```

### 端口被占用
```bash
# 查看端口占用
lsof -i :5000

# 使用其他端口
./start_panel.sh -p 8080
```

## 📞 技术支持

- **GitHub**: https://github.com/wudi2898/cc-main
- **问题反馈**: 提交Issue
- **功能建议**: 提交Pull Request

---

**🎉 安装优化完成！现在可以一键安装、启动和管理CC压测工具了！**
