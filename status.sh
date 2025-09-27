#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}📊 CC压力测试服务状态${NC}"
echo ""

# 检查API服务器状态
API_PIDS=$(pgrep -f "api_server")
if [ -n "$API_PIDS" ]; then
    echo -e "${GREEN}✅ API服务器运行中${NC}"
    for pid in $API_PIDS; do
        echo -e "   PID: $pid"
        echo -e "   启动时间: $(ps -o lstart= -p $pid)"
        echo -e "   内存使用: $(ps -o rss= -p $pid | awk '{print $1/1024 " MB"}')"
        echo -e "   CPU使用: $(ps -o %cpu= -p $pid)%"
    done
else
    echo -e "${RED}❌ API服务器未运行${NC}"
fi

echo ""

# 检查主程序状态
MAIN_PIDS=$(pgrep -f "cc-go")
if [ -n "$MAIN_PIDS" ]; then
    echo -e "${GREEN}✅ 主程序运行中${NC}"
    for pid in $MAIN_PIDS; do
        echo -e "   PID: $pid"
        echo -e "   启动时间: $(ps -o lstart= -p $pid)"
        echo -e "   内存使用: $(ps -o rss= -p $pid | awk '{print $1/1024 " MB"}')"
        echo -e "   CPU使用: $(ps -o %cpu= -p $pid)%"
    done
else
    echo -e "${YELLOW}⚠️  主程序未运行${NC}"
fi

echo ""

# 检查端口占用
echo -e "${BLUE}🔍 端口状态:${NC}"
if netstat -tuln 2>/dev/null | grep -q ":8080 "; then
    echo -e "${GREEN}✅ 端口8080已被占用${NC}"
    netstat -tuln | grep ":8080 "
else
    echo -e "${RED}❌ 端口8080未被占用${NC}"
fi

echo ""

# 检查日志文件
echo -e "${BLUE}📋 日志文件:${NC}"
if [ -f "api_server.log" ]; then
    echo -e "${GREEN}✅ api_server.log 存在${NC}"
    echo -e "   文件大小: $(ls -lh api_server.log | awk '{print $5}')"
    echo -e "   最后修改: $(ls -l api_server.log | awk '{print $6, $7, $8}')"
else
    echo -e "${YELLOW}⚠️  api_server.log 不存在${NC}"
fi

echo ""
echo -e "${BLUE}💡 管理命令:${NC}"
echo -e "   启动服务: ./start.sh"
echo -e "   停止服务: ./stop.sh"
echo -e "   查看日志: tail -f api_server.log"
echo -e "   访问前端: http://localhost:8080"
