#!/usr/bin/python3
"""
代理类型性能对比测试
对比SOCKS5和HTTP代理的性能差异
"""

import time
import threading
import socket
import socks
import ssl
import statistics
from concurrent.futures import ThreadPoolExecutor, as_completed

class ProxyBenchmark:
    def __init__(self):
        self.results = {
            'socks5': [],
            'http': []
        }
    
    def test_socks5_connection(self, proxy_host, proxy_port, target_host, target_port):
        """测试SOCKS5代理连接性能"""
        start_time = time.time()
        
        try:
            s = socks.socksocket()
            s.set_proxy(socks.SOCKS5, proxy_host, proxy_port)
            s.settimeout(10)
            s.connect((target_host, target_port))
            
            # 发送简单HTTP请求
            request = f"GET / HTTP/1.1\r\nHost: {target_host}\r\nConnection: close\r\n\r\n"
            s.send(request.encode())
            
            # 记录发送完成时间
            send_time = time.time()
            
            s.close()
            return send_time - start_time
            
        except Exception as e:
            return None
    
    def test_http_proxy_connection(self, proxy_host, proxy_port, target_host, target_port):
        """测试HTTP代理连接性能"""
        start_time = time.time()
        
        try:
            # 连接到HTTP代理
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.settimeout(10)
            s.connect((proxy_host, proxy_port))
            
            # 发送CONNECT请求
            connect_request = f"CONNECT {target_host}:{target_port} HTTP/1.1\r\nHost: {target_host}:{target_port}\r\n\r\n"
            s.send(connect_request.encode())
            
            # 接收CONNECT响应
            response = s.recv(1024).decode()
            if "200 Connection established" not in response:
                s.close()
                return None
            
            # 发送实际HTTP请求
            request = f"GET / HTTP/1.1\r\nHost: {target_host}\r\nConnection: close\r\n\r\n"
            s.send(request.encode())
            
            # 记录发送完成时间
            send_time = time.time()
            
            s.close()
            return send_time - start_time
            
        except Exception as e:
            return None
    
    def run_benchmark(self, proxy_type, proxy_host, proxy_port, target_host="httpbin.org", target_port=80, num_tests=50):
        """运行基准测试"""
        print(f"\n测试 {proxy_type.upper()} 代理性能...")
        print(f"代理: {proxy_host}:{proxy_port}")
        print(f"目标: {target_host}:{target_port}")
        print(f"测试次数: {num_tests}")
        
        successful_tests = 0
        failed_tests = 0
        times = []
        
        with ThreadPoolExecutor(max_workers=10) as executor:
            # 提交所有测试任务
            futures = []
            for i in range(num_tests):
                if proxy_type == 'socks5':
                    future = executor.submit(self.test_socks5_connection, proxy_host, proxy_port, target_host, target_port)
                else:  # http
                    future = executor.submit(self.test_http_proxy_connection, proxy_host, proxy_port, target_host, target_port)
                futures.append(future)
            
            # 收集结果
            for i, future in enumerate(as_completed(futures)):
                result = future.result()
                if result is not None:
                    times.append(result)
                    successful_tests += 1
                    print(f"\r进度: {i+1}/{num_tests}, 成功: {successful_tests}, 失败: {failed_tests}", end="")
                else:
                    failed_tests += 1
                    print(f"\r进度: {i+1}/{num_tests}, 成功: {successful_tests}, 失败: {failed_tests}", end="")
        
        print("\n")
        
        if times:
            avg_time = statistics.mean(times)
            min_time = min(times)
            max_time = max(times)
            median_time = statistics.median(times)
            
            print(f"结果统计:")
            print(f"  成功率: {successful_tests/num_tests*100:.1f}% ({successful_tests}/{num_tests})")
            print(f"  平均响应时间: {avg_time:.3f}s")
            print(f"  最快响应时间: {min_time:.3f}s")
            print(f"  最慢响应时间: {max_time:.3f}s")
            print(f"  中位响应时间: {median_time:.3f}s")
            
            self.results[proxy_type] = {
                'success_rate': successful_tests/num_tests*100,
                'avg_time': avg_time,
                'min_time': min_time,
                'max_time': max_time,
                'median_time': median_time,
                'total_tests': num_tests,
                'successful_tests': successful_tests
            }
        else:
            print("所有测试都失败了！")
            self.results[proxy_type] = None
    
    def compare_results(self):
        """对比测试结果"""
        print(f"\n{'='*60}")
        print("代理类型性能对比")
        print(f"{'='*60}")
        
        if self.results['socks5'] and self.results['http']:
            socks5 = self.results['socks5']
            http = self.results['http']
            
            print(f"{'指标':<20} {'SOCKS5':<15} {'HTTP':<15} {'优势'}")
            print(f"{'-'*60}")
            
            # 成功率对比
            socks5_success = socks5['success_rate']
            http_success = http['success_rate']
            success_winner = "SOCKS5" if socks5_success > http_success else "HTTP" if http_success > socks5_success else "相同"
            print(f"{'成功率':<20} {socks5_success:<15.1f}% {http_success:<15.1f}% {success_winner}")
            
            # 平均响应时间对比
            socks5_avg = socks5['avg_time']
            http_avg = http['avg_time']
            speed_winner = "SOCKS5" if socks5_avg < http_avg else "HTTP"
            speed_improvement = abs(socks5_avg - http_avg) / max(socks5_avg, http_avg) * 100
            print(f"{'平均响应时间':<20} {socks5_avg:<15.3f}s {http_avg:<15.3f}s {speed_winner} (+{speed_improvement:.1f}%)")
            
            # 最快响应时间对比
            socks5_min = socks5['min_time']
            http_min = http['min_time']
            min_winner = "SOCKS5" if socks5_min < http_min else "HTTP"
            print(f"{'最快响应时间':<20} {socks5_min:<15.3f}s {http_min:<15.3f}s {min_winner}")
            
            # 稳定性对比 (通过标准差体现)
            socks5_range = socks5['max_time'] - socks5['min_time']
            http_range = http['max_time'] - http['min_time']
            stability_winner = "SOCKS5" if socks5_range < http_range else "HTTP"
            print(f"{'响应时间范围':<20} {socks5_range:<15.3f}s {http_range:<15.3f}s {stability_winner}")
            
            print(f"\n结论:")
            if socks5_avg < http_avg and socks5_success >= http_success:
                improvement = (http_avg - socks5_avg) / http_avg * 100
                print(f"✅ SOCKS5代理性能明显优于HTTP代理")
                print(f"   - 响应时间提升: {improvement:.1f}%")
                print(f"   - 建议使用SOCKS5代理以获得最佳性能")
            elif http_avg < socks5_avg:
                improvement = (socks5_avg - http_avg) / socks5_avg * 100
                print(f"⚠️  HTTP代理性能略优于SOCKS5代理")
                print(f"   - 响应时间提升: {improvement:.1f}%")
                print(f"   - 可能是代理质量差异导致")
            else:
                print(f"📊 两种代理性能相近")
                print(f"   - 选择时可考虑其他因素（如隐蔽性、协议支持等）")
                
        else:
            print("缺少完整的测试数据，无法进行对比")

def main():
    """主函数"""
    print("代理类型性能对比测试")
    print("此测试将对比SOCKS5和HTTP代理的连接性能")
    
    # 获取用户输入
    print("\n请提供测试代理信息:")
    
    # SOCKS5代理
    socks5_host = input("SOCKS5代理地址 (默认: 127.0.0.1): ").strip() or "127.0.0.1"
    socks5_port = int(input("SOCKS5代理端口 (默认: 1080): ").strip() or "1080")
    
    # HTTP代理
    http_host = input("HTTP代理地址 (默认: 127.0.0.1): ").strip() or "127.0.0.1"
    http_port = int(input("HTTP代理端口 (默认: 8080): ").strip() or "8080")
    
    # 测试目标
    target_host = input("测试目标 (默认: httpbin.org): ").strip() or "httpbin.org"
    target_port = int(input("目标端口 (默认: 80): ").strip() or "80")
    
    # 测试次数
    num_tests = int(input("测试次数 (默认: 50): ").strip() or "50")
    
    # 创建基准测试实例
    benchmark = ProxyBenchmark()
    
    # 运行测试
    try:
        # 测试SOCKS5
        benchmark.run_benchmark('socks5', socks5_host, socks5_port, target_host, target_port, num_tests)
        
        # 等待一下
        time.sleep(2)
        
        # 测试HTTP代理
        benchmark.run_benchmark('http', http_host, http_port, target_host, target_port, num_tests)
        
        # 对比结果
        benchmark.compare_results()
        
    except KeyboardInterrupt:
        print("\n测试被用户中断")
    except Exception as e:
        print(f"\n测试出错: {e}")

if __name__ == "__main__":
    main()
