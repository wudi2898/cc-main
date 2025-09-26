#!/bin/bash
echo "正在安装DDoS压测工具依赖..."

# 检查Python版本
python_version=$(python3 --version 2>&1 | grep -o '[0-9]\+\.[0-9]\+' | head -1)
echo "检测到Python版本: $python_version"

# 安装pip依赖
echo "安装Python依赖包..."
pip3 install -r requirements.txt

# 创建必要目录
mkdir -p logs
mkdir -p static
mkdir -p templates

# 设置权限
chmod +x start.sh
chmod +x stop.sh
chmod +x web_panel.py
chmod +x main.py

echo "安装完成！"
echo ""
echo "使用方法："
echo "1. 命令行版本："
echo "   python3 main.py cc https://example.com 100 10 --cf-bypass"
echo ""
echo "2. Web控制面板："
echo "   python3 web_panel.py"
echo "   然后访问 http://localhost:5000"
echo ""
echo "注意: 请先在 socks5.txt 中配置代理列表"
