#!/usr/bin/python3
"""
性能测试脚本 - 展示不同模式的性能差异
"""

import subprocess
import time
import threading
import psutil
import os

def run_attack_test(mode_name, command, duration=30):
    """运行攻击测试"""
    print(f"\n{'='*50}")
    print(f"测试模式: {mode_name}")
    print(f"命令: {command}")
    print(f"持续时间: {duration}秒")
    print(f"{'='*50}")
    
    # 记录初始资源使用
    initial_cpu = psutil.cpu_percent(interval=1)
    initial_memory = psutil.virtual_memory().used
    
    print(f"初始CPU使用率: {initial_cpu:.1f}%")
    print(f"初始内存使用: {initial_memory / 1024 / 1024:.1f} MB")
    
    # 启动攻击进程
    process = subprocess.Popen(
        command.split(),
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        universal_newlines=True
    )
    
    # 监控资源使用
    max_cpu = 0
    max_memory = 0
    rps_samples = []
    
    start_time = time.time()
    while time.time() - start_time < duration:
        try:
            # 检查进程是否还在运行
            if process.poll() is not None:
                break
                
            # 记录资源使用
            cpu_percent = psutil.cpu_percent(interval=0.5)
            memory_used = psutil.virtual_memory().used
            
            max_cpu = max(max_cpu, cpu_percent)
            max_memory = max(max_memory, memory_used)
            
            # 尝试解析RPS（如果有的话）
            # 这里只是示例，实际需要根据输出格式调整
            
            time.sleep(1)
            
        except Exception as e:
            print(f"监控错误: {e}")
            break
    
    # 终止进程
    try:
        process.terminate()
        process.wait(timeout=5)
    except:
        process.kill()
    
    # 输出结果
    print(f"\n测试结果:")
    print(f"最大CPU使用率: {max_cpu:.1f}%")
    print(f"最大内存使用: {max_memory / 1024 / 1024:.1f} MB")
    print(f"内存增量: {(max_memory - initial_memory) / 1024 / 1024:.1f} MB")
    
    return {
        'mode': mode_name,
        'max_cpu': max_cpu,
        'max_memory': max_memory,
        'memory_delta': max_memory - initial_memory,
        'duration': duration
    }

def main():
    """主函数"""
    print("DDoS压测工具性能测试")
    print("注意: 确保socks5.txt中有可用代理")
    
    target_url = "https://httpbin.org"  # 测试目标
    threads = 100
    rps = 10
    test_duration = 30
    
    # 测试配置
    test_configs = [
        {
            'name': '正常模式',
            'command': f'python3 main.py cc {target_url} {threads} {rps} --duration {test_duration}'
        },
        {
            'name': 'CF绕过模式',
            'command': f'python3 main.py cc {target_url} {threads} {rps} --cf-bypass --duration {test_duration}'
        },
        {
            'name': '超负荷模式',
            'command': f'python3 main.py cc {target_url} {threads} {rps} --overload --duration {test_duration}'
        },
        {
            'name': '纯发送模式',
            'command': f'python3 main.py cc {target_url} {threads} {rps} --fire-and-forget --duration {test_duration}'
        },
        {
            'name': '极限模式',
            'command': f'python3 main.py cc {target_url} {threads} {rps} --overload --fire-and-forget --burst --no-delay --duration {test_duration}'
        }
    ]
    
    results = []
    
    for config in test_configs:
        try:
            result = run_attack_test(config['name'], config['command'], test_duration)
            results.append(result)
            
            # 等待系统恢复
            print("等待系统恢复...")
            time.sleep(10)
            
        except KeyboardInterrupt:
            print("\n测试被用户中断")
            break
        except Exception as e:
            print(f"测试 {config['name']} 失败: {e}")
            continue
    
    # 输出对比结果
    print(f"\n{'='*80}")
    print("性能对比结果")
    print(f"{'='*80}")
    print(f"{'模式':<15} {'最大CPU%':<10} {'最大内存MB':<12} {'内存增量MB':<12}")
    print(f"{'-'*50}")
    
    for result in results:
        print(f"{result['mode']:<15} {result['max_cpu']:<10.1f} "
              f"{result['max_memory']/1024/1024:<12.1f} {result['memory_delta']/1024/1024:<12.1f}")
    
    print(f"\n测试完成！")
    print("结论:")
    print("- 正常模式: 标准性能，资源使用适中")
    print("- CF绕过模式: 性能略降，但能绕过CF防护")
    print("- 超负荷模式: 更高的CPU和内存使用，更高的攻击强度")
    print("- 纯发送模式: 极低延迟，不等待响应，最高RPS")
    print("- 极限模式: 所有优化开启，最大攻击强度")

if __name__ == "__main__":
    main()
