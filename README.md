# 高级压力测试工具 - Go版本

🚀 高性能的HTTP压力测试工具，专门针对SOCKS5代理和Cloudflare绕过优化。

## ✨ 特性

- **高性能**: 基于Go语言，支持数万并发连接
- **SOCKS5代理**: 支持SOCKS5代理池，自动轮换
- **CF绕过**: 专门针对Cloudflare防护优化，上亿万万个组合
- **Web控制面板**: 现代化Web界面，实时监控
- **前后端分离**: 纯HTML+JavaScript前端，Go API后端
- **实时统计**: 实时显示请求统计和RPS
- **一键启动**: 自动构建和启动

## 🚀 快速开始

### 一行命令安装运行

```bash
# 一行命令安装并运行（自动安装Go、构建、启动）
curl -fsSL https://raw.githubusercontent.com/your-repo/cc-main/main/install.sh | bash
```

### 手动安装（可选）

```bash
# 克隆项目
git clone https://github.com/your-repo/cc-main.git
cd cc-main

# 手动安装
chmod +x install.sh
./install.sh
```

### 访问控制面板

- **主页面**: http://localhost:8080
- **日志页面**: http://localhost:8080/logs.html
- **API接口**: http://localhost:8080/api

### 一行命令自定义配置

```bash
# 自定义端口
curl -fsSL https://raw.githubusercontent.com/your-repo/cc-main/main/install.sh | bash -s -- -p 9090

# 自定义任务文件路径
curl -fsSL https://raw.githubusercontent.com/your-repo/cc-main/main/install.sh | bash -s -- -t /my-tasks.json

# 查看帮助
curl -fsSL https://raw.githubusercontent.com/your-repo/cc-main/main/install.sh | bash -s -- -h
```

### 其他一行命令方式

```bash
# 使用wget
wget -qO- https://raw.githubusercontent.com/your-repo/cc-main/main/install.sh | bash

# 使用curl（推荐）
curl -fsSL https://raw.githubusercontent.com/your-repo/cc-main/main/install.sh | bash
```

## 🎯 使用方法

### Web控制面板

1. 打开浏览器访问 http://localhost:8080
2. 点击"创建新任务"按钮
3. 填写目标URL和参数
4. 点击"创建任务"并启动
5. 实时查看任务状态和日志

### 命令行使用

```bash
# 构建主程序
go build -o cc-go main.go

# 基本用法
./cc-go -u https://example.com -t 1000 -r 5000 -d 60

# 使用SOCKS5代理
./cc-go -u https://example.com -t 1000 -r 5000 -d 60 -proxy-file socks5.txt

# 启用CF绕过
./cc-go -u https://example.com -t 1000 -r 5000 -d 60 -cf-bypass
```

## 📊 性能配置

### 推荐配置

| 场景 | 线程数 | RPS | 说明 |
|------|--------|-----|------|
| 轻量测试 | 1,000 | 5,000 | 适合小目标 |
| 中等测试 | 10,000 | 50,000 | 适合中等目标 |
| 高强度测试 | 50,000 | 200,000 | 适合大目标 |
| 极限测试 | 100,000 | 500,000 | 适合高防目标 |

### 参数说明

- `-u`: 目标URL
- `-t`: 线程数 (1-100,000)
- `-r`: 每秒请求数 (1-1,000,000)
- `-d`: 持续时间(秒) (1-86400)
- `-timeout`: 超时时间(秒) (1-300)
- `-cf-bypass`: 启用CF绕过
- `-random-path`: 随机路径
- `-random-params`: 随机参数

## 🛡️ CF绕过特性 - 上亿万万个组合

- **随机User-Agent**: 使用第三方库生成真实浏览器UA，无限组合
- **随机Referer**: 完全随机生成，包含50+域名和30+路径组合
- **随机HTTP头**: 每次请求随机生成5-15个不同的HTTP头
- **随机路径和参数**: 动态生成随机URL路径和查询参数
- **高级HTTP头**: 包含Sec-Ch-Ua、Sec-Fetch等现代浏览器头
- **CF特殊头模拟**: 模拟Cloudflare相关头信息
- **完全随机化**: 每次请求都不同，实现上亿万万个组合
- **无配置文件**: 完全基于算法生成，无需外部文件

## 📊 实时统计

工具会实时显示：
- 总请求数
- 成功请求数
- 失败请求数
- 当前RPS
- 平均RPS
- 运行时间

## 📁 文件结构

```
cc-main/
├── main.go              # 核心攻击程序
├── api_server.go        # Web API服务器
├── go.mod               # Go模块文件
├── install.sh           # 一键安装脚本
├── start.sh             # 快速启动脚本
├── socks5.txt           # SOCKS5代理文件（可选）
├── /cc-tasks.json       # 任务列表配置文件
├── frontend/            # 前端文件
│   ├── index.html       # 主页面
│   ├── logs.html        # 日志页面
│   ├── css/             # 样式文件
│   ├── js/              # JavaScript文件
│   └── fonts/           # 字体文件
└── README.md            # 说明文档
```

## 🔧 开发

### 构建

```bash
# 构建主程序
go build -o cc-go main.go

# 构建API服务器
go build -o api_server api_server.go

# 交叉编译 (Linux)
GOOS=linux GOARCH=amd64 go build -o cc-go main.go

# 交叉编译 (Windows)
GOOS=windows GOARCH=amd64 go build -o cc-go.exe main.go
```

## ⚠️ 免责声明

本工具仅用于学习和测试目的，请勿用于非法用途。使用者需遵守当地法律法规，作者不承担任何责任。