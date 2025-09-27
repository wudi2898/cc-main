#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🛑 停止CC压力测试服务...${NC}"

# 查找并停止API服务器进程
API_PIDS=$(pgrep -f "api_server")
if [ -n "$API_PIDS" ]; then
    echo -e "${YELLOW}📋 找到API服务器进程: $API_PIDS${NC}"
    for pid in $API_PIDS; do
        echo -e "${GREEN}🔄 正在停止进程 $pid...${NC}"
        kill $pid
        sleep 1
        
        # 检查进程是否已停止
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${YELLOW}⚠️  进程 $pid 未响应，强制停止...${NC}"
            kill -9 $pid
        fi
    done
    echo -e "${GREEN}✅ API服务器已停止${NC}"
else
    echo -e "${YELLOW}⚠️  未找到运行中的API服务器进程${NC}"
fi

# 查找并停止主程序进程
MAIN_PIDS=$(pgrep -f "cc-go")
if [ -n "$MAIN_PIDS" ]; then
    echo -e "${YELLOW}📋 找到主程序进程: $MAIN_PIDS${NC}"
    for pid in $MAIN_PIDS; do
        echo -e "${GREEN}🔄 正在停止进程 $pid...${NC}"
        kill $pid
        sleep 1
        
        # 检查进程是否已停止
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${YELLOW}⚠️  进程 $pid 未响应，强制停止...${NC}"
            kill -9 $pid
        fi
    done
    echo -e "${GREEN}✅ 主程序已停止${NC}"
else
    echo -e "${YELLOW}⚠️  未找到运行中的主程序进程${NC}"
fi

echo -e "${BLUE}🎉 所有服务已停止${NC}"
