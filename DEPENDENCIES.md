# 📦 依赖说明

## 核心依赖

### Web框架
- **flask==2.3.3** - Web框架，用于Web控制面板
- **flask-socketio==5.3.6** - WebSocket支持，用于实时通信

### 系统监控
- **psutil==5.9.5** - 系统资源监控（CPU、内存、网络）

### 代理支持
- **PySocks==1.7.1** - SOCKS4/5代理支持

## 可选依赖

### 网络请求
- **requests==2.31.0** - HTTP请求库，用于代理测试

### 加密支持
- **cryptography==41.0.7** - 现代加密算法支持

### 异步支持
- **eventlet==0.33.3** - 异步网络库，提升性能

### 日志处理
- **colorlog==6.7.0** - 彩色日志输出

### 配置管理
- **pyyaml==6.0.1** - YAML配置文件支持

### 时间处理
- **python-dateutil==2.8.2** - 时间解析和处理

### 数据验证
- **jsonschema==4.19.2** - JSON数据验证

### 性能优化
- **uvloop==0.19.0** - 高性能事件循环（仅Linux/macOS）

## 安装说明

### 完整安装
```bash
pip install -r requirements.txt
```

### 最小安装（仅核心功能）
```bash
pip install flask==2.3.3 flask-socketio==5.3.6 psutil==5.9.5 PySocks==1.7.1
```

### 开发环境安装
```bash
pip install -r requirements.txt
pip install pytest==7.4.3 black==23.9.1 flake8==6.1.0
```

## 版本兼容性

### Python版本
- **最低要求**: Python 3.7+
- **推荐版本**: Python 3.9+
- **测试版本**: Python 3.11

### 操作系统支持
- **Linux**: 完全支持
- **macOS**: 完全支持
- **Windows**: 部分支持（无uvloop）

## 依赖冲突解决

### 常见问题

1. **Flask版本冲突**
   ```bash
   pip install --upgrade flask flask-socketio
   ```

2. **PySocks安装失败**
   ```bash
   pip install --upgrade pip
   pip install PySocks
   ```

3. **psutil权限问题**
   ```bash
   pip install --user psutil
   ```

### 虚拟环境推荐
```bash
# 创建虚拟环境
python3 -m venv venv

# 激活虚拟环境
source venv/bin/activate  # Linux/macOS
# 或
venv\Scripts\activate     # Windows

# 安装依赖
pip install -r requirements.txt
```

## 性能优化建议

### 生产环境
- 使用 `eventlet` 作为WSGI服务器
- 启用 `uvloop`（Linux/macOS）
- 配置适当的日志级别

### 开发环境
- 启用Flask调试模式
- 使用彩色日志输出
- 安装代码质量工具

## 安全注意事项

- 定期更新依赖包
- 检查安全漏洞：`pip audit`
- 使用虚拟环境隔离
- 避免在生产环境使用调试模式
