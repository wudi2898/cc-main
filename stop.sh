#!/bin/bash

for pid in $(pgrep -f 'main.py cc xxxxxxx'); do  # 使用 pgrep -f 来精确匹配整个命令
    echo "Force stopping process with PID: $pid"
    
    # 强制终止进程
    kill -9 "$pid"
    
    # 打印已停止进程的信息
    if ! ps -p "$pid" > /dev/null; then
        echo "Process $pid 已被强制停止"
    else
        echo "Failed to stop process $pid"
    fi
done

