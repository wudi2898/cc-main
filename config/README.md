# 📁 配置文件说明

## 文件结构

```
config/
├── socks5.txt          # SOCKS5代理列表
├── http_proxies.txt    # HTTP代理列表
├── accept_headers.txt  # HTTP请求头列表
├── referers.txt        # 引用页列表
└── README.md          # 配置文件说明
```

## 配置文件说明

### 1. socks5.txt - SOCKS5代理列表
```
# SOCKS5代理列表 - 每行一个代理
# 格式: IP:端口
# 示例:
127.0.0.1:1080
192.168.1.100:7890
proxy.example.com:1080
```

**要求**:
- 每行一个代理
- 格式: `IP:端口`
- 支持注释行（以#开头）
- 至少需要1个有效代理

### 2. http_proxies.txt - HTTP代理列表
```
# HTTP代理列表 - 每行一个代理
# 格式: IP:端口
# 示例:
127.0.0.1:8080
192.168.1.100:3128
proxy.example.com:8080
```

**要求**:
- 每行一个代理
- 格式: `IP:端口`
- 支持注释行（以#开头）
- 可选配置

### 3. accept_headers.txt - HTTP请求头列表
```
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
Accept-Language: en-US,en;q=0.5
Accept-Encoding: gzip, deflate
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8
```

**要求**:
- 每行一个请求头
- 格式: `Header-Name: Header-Value`
- 支持注释行（以#开头）
- 可选配置（有默认值）

### 4. referers.txt - 引用页列表
```
https://www.google.com/search?q=
https://www.facebook.com/
https://www.youtube.com/
https://www.bing.com/search?q=
https://www.baidu.com/s?wd=
```

**要求**:
- 每行一个引用页
- 格式: `https://example.com/path`
- 支持注释行（以#开头）
- 可选配置（有默认值）

## 配置建议

### 代理质量要求
- **延迟**: < 500ms
- **稳定性**: > 95%
- **带宽**: > 10Mbps
- **地理分布**: 多国IP

### 请求头建议
- 使用现代浏览器特征
- 包含完整的Accept系列头
- 模拟真实用户行为

### 引用页建议
- 使用知名网站
- 包含搜索参数
- 模拟真实来源

## 编辑配置文件

### 方法1: 命令行编辑
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

### 方法2: Web面板编辑
通过Web控制面板的配置页面进行编辑。

### 方法3: 直接替换
```bash
# 替换SOCKS5代理
echo "127.0.0.1:1080" > config/socks5.txt

# 添加多个代理
cat > config/socks5.txt << EOF
127.0.0.1:1080
192.168.1.100:7890
proxy.example.com:1080
EOF
```

## 验证配置

### 测试代理连接
```bash
# 测试SOCKS5代理
python3 main.py check https://httpbin.org/ip 1 1

# 测试HTTP代理
python3 main.py check https://httpbin.org/ip 1 1 --proxy-type http
```

### 检查配置文件
```bash
# 查看代理数量
wc -l config/socks5.txt

# 查看有效代理
grep -v "^#" config/socks5.txt | wc -l
```

## 注意事项

⚠️ **重要提醒**:
- 确保代理服务器稳定可靠
- 定期更新代理列表
- 遵守代理服务商的使用条款
- 仅用于授权的安全测试
