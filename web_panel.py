#!/usr/bin/python3
"""
DDoS压测工具Web控制面板
功能:
1. 任务管理和定时启动
2. 实时日志查看
3. 系统性能监控
4. 压测时长控制
"""

import os
import sys
import json
import time
import psutil
import signal
import threading
import subprocess
from datetime import datetime, timedelta
# 依赖检查
try:
    from flask import Flask, render_template, request, jsonify, Response
    from flask_socketio import SocketIO, emit
    FLASK_AVAILABLE = True
except ImportError:
    FLASK_AVAILABLE = False
    print("错误: Flask相关依赖未安装")
    print("安装命令: pip install flask flask-socketio psutil")
    sys.exit(1)
import logging
from logging.handlers import RotatingFileHandler

app = Flask(__name__)
app.config['SECRET_KEY'] = 'ddos_panel_secret_key_2024'
socketio = SocketIO(app, cors_allowed_origins="*")

# 全局变量
running_tasks = {}
task_logs = {}
system_stats = {}

class TaskManager:
    def __init__(self):
        self.tasks = {}
        self.task_counter = 0
        
    def create_task(self, config):
        """创建新任务"""
        task_id = f"task_{int(time.time())}_{self.task_counter}"
        self.task_counter += 1
        
        task = {
            'id': task_id,
            'config': config,
            'status': 'created',
            'pid': None,
            'start_time': None,
            'end_time': None,
            'duration': config.get('duration', 0),
            'auto_restart': config.get('auto_restart', False),
            'restart_interval': config.get('restart_interval', 60),  # 重启间隔（秒）
            'logs': []
        }
        
        self.tasks[task_id] = task
        return task_id
        
    def start_task(self, task_id):
        """启动任务"""
        if task_id not in self.tasks:
            return False
            
        task = self.tasks[task_id]
        if task['status'] == 'running':
            return False
        
        # 如果任务已完成，重置状态
        if task['status'] == 'completed':
            task['status'] = 'created'
            task['pid'] = None
            task['start_time'] = None
            task['end_time'] = None
            task['process'] = None
            
        # 构建命令
        config = task['config']
        # 获取项目目录
        if os.path.exists('/opt/cc-main'):
            project_dir = '/opt/cc-main'
        else:
            project_dir = os.getcwd()
        
        python_path = os.path.join(project_dir, 'venv', 'bin', 'python')
        main_py_path = os.path.join(project_dir, 'main.py')
        
        # 检查文件是否存在
        if not os.path.exists(main_py_path):
            task['logs'].append({
                'timestamp': datetime.now().strftime('%H:%M:%S'),
                'message': f'错误: main.py 文件不存在于 {main_py_path}'
            })
            return False
        
        cmd = [
            python_path, main_py_path,
            config['mode'],
            config['url'],
            str(config['threads']),
            str(config['rps']),
            '--proxy-file', 'config/socks5.txt'
        ]
        
        if config.get('cookies'):
            cmd.extend(['--cookies', config['cookies']])
        if config.get('timeout'):
            cmd.extend(['--timeout', str(config['timeout'])])
            
        try:
            # 调试信息
            print(f"启动任务: {task_id}")
            print(f"命令: {' '.join(cmd)}")
            print(f"工作目录: {project_dir}")
            print(f"Python路径: {python_path}")
            print(f"main.py路径: {main_py_path}")
            
            # 启动进程
            process = subprocess.Popen(
                cmd,
                cwd=project_dir,  # 使用项目目录
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                universal_newlines=True,
                bufsize=1
            )
            
            task['status'] = 'running'
            task['pid'] = process.pid
            task['start_time'] = datetime.now()
            task['process'] = process
            
            # 启动日志读取线程
            log_thread = threading.Thread(
                target=self._read_task_logs,
                args=(task_id, process),
                daemon=True
            )
            log_thread.start()
            
            # 启动时长控制线程
            if task['duration'] > 0:
                timer_thread = threading.Thread(
                    target=self._task_timer,
                    args=(task_id, task['duration']),
                    daemon=True
                )
                timer_thread.start()
                
            return True
            
        except Exception as e:
            task['status'] = 'error'
            task['logs'].append(f"启动失败: {str(e)}")
            return False
            
    def stop_task(self, task_id):
        """停止任务"""
        if task_id not in self.tasks:
            return False
            
        task = self.tasks[task_id]
        if task['status'] != 'running':
            return False
            
        try:
            if 'process' in task:
                task['process'].terminate()
                task['process'].wait(timeout=5)
            elif task['pid']:
                os.kill(task['pid'], signal.SIGTERM)
                
            task['status'] = 'stopped'
            task['end_time'] = datetime.now()
            return True
            
        except Exception as e:
            task['logs'].append(f"停止失败: {str(e)}")
            return False
            
    def _read_task_logs(self, task_id, process):
        """读取任务日志 - 改进的实时日志读取"""
        task = self.tasks[task_id]
        
        try:
            import select
            import sys
            import fcntl
            
            # 设置非阻塞模式
            if sys.platform != 'win32':
                fd = process.stdout.fileno()
                fl = fcntl.fcntl(fd, fcntl.F_GETFL)
                fcntl.fcntl(fd, fcntl.F_SETFL, fl | os.O_NONBLOCK)
            
            while True:
                # 检查进程是否还在运行
                if process.poll() is not None:
                    break
                    
                try:
                    # 非阻塞读取
                    if sys.platform != 'win32':
                        # Unix系统使用select
                        ready, _, _ = select.select([process.stdout], [], [], 0.05)
                        if ready:
                            line = process.stdout.readline()
                            if line:
                                self._add_log_entry(task_id, line.strip())
                    else:
                        # Windows系统
                        line = process.stdout.readline()
                        if line:
                            self._add_log_entry(task_id, line.strip())
                        else:
                            time.sleep(0.05)
                            
                except (OSError, IOError):
                    # 没有数据可读，继续循环
                    time.sleep(0.05)
                    continue
                        
            # 读取剩余输出
            try:
                remaining_output = process.stdout.read()
                if remaining_output:
                    for line in remaining_output.splitlines():
                        if line.strip():
                            self._add_log_entry(task_id, line.strip())
            except:
                pass
            
            # 等待进程结束
            return_code = process.wait()
            
            # 检查任务是否还在运行状态
            task = self.tasks.get(task_id)
            if task and task['status'] == 'running':
                task['status'] = 'completed'
                task['end_time'] = datetime.now()
                self._add_log_entry(task_id, f'任务完成，退出码: {return_code}')
                
                # 如果启用了自动重启
                if task.get('auto_restart', False):
                    restart_interval = task.get('restart_interval', 60)
                    self._add_log_entry(task_id, f'任务将在 {restart_interval} 秒后自动重启')
                    
                    # 启动重启定时器
                    restart_thread = threading.Thread(
                        target=self._auto_restart_task,
                        args=(task_id, restart_interval),
                        daemon=True
                    )
                    restart_thread.start()
                
        except Exception as e:
            self._add_log_entry(task_id, f"日志读取错误: {str(e)}")
    
    def _add_log_entry(self, task_id, message):
        """添加日志条目并实时推送"""
        task = self.tasks.get(task_id)
        if not task:
            return
            
        # 清理和格式化消息
        message = message.strip()
        if not message:
            return
            
        # 解析日志级别和内容
        log_level = "INFO"
        if "ERROR" in message or "错误" in message or "失败" in message:
            log_level = "ERROR"
        elif "WARNING" in message or "警告" in message:
            log_level = "WARNING"
        elif "INFO" in message or "信息" in message:
            log_level = "INFO"
        elif "DEBUG" in message or "调试" in message:
            log_level = "DEBUG"
            
        log_entry = {
            'timestamp': datetime.now().strftime('%H:%M:%S'),
            'level': log_level,
            'message': message
        }
        
        task['logs'].append(log_entry)
        
        # 限制日志数量
        if len(task['logs']) > 1000:
            task['logs'] = task['logs'][-500:]
            
        # 实时推送日志
        try:
            socketio.emit('task_log', {
                'task_id': task_id,
                'log': log_entry
            })
            print(f"推送日志: {task_id} - {message}")  # 调试信息
        except Exception as e:
            print(f"WebSocket推送失败: {e}")
            
    def _task_timer(self, task_id, duration):
        """任务时长控制"""
        time.sleep(duration)
        
        task = self.tasks[task_id]
        if task['status'] == 'running':
            self.stop_task(task_id)
            task['logs'].append({
                'timestamp': datetime.now().strftime('%H:%M:%S'),
                'message': f"任务已运行 {duration} 秒，自动停止"
            })
    
    def _auto_restart_task(self, task_id, interval):
        """自动重启任务"""
        time.sleep(interval)
        
        task = self.tasks.get(task_id)
        if task and task['status'] == 'completed':
            task['logs'].append({
                'timestamp': datetime.now().strftime('%H:%M:%S'),
                'message': '开始自动重启任务'
            })
            self.start_task(task_id)

# 初始化任务管理器
task_manager = TaskManager()

def get_system_stats():
    """获取系统性能统计"""
    try:
        stats = {
            'timestamp': datetime.now().strftime('%H:%M:%S'),
            'cpu_percent': psutil.cpu_percent(interval=1),
            'memory': {
                'total': psutil.virtual_memory().total,
                'available': psutil.virtual_memory().available,
                'percent': psutil.virtual_memory().percent,
                'used': psutil.virtual_memory().used
            },
            'disk': {
                'total': psutil.disk_usage('/').total,
                'used': psutil.disk_usage('/').used,
                'free': psutil.disk_usage('/').free,
                'percent': psutil.disk_usage('/').percent
            },
            'network': {
                'bytes_sent': psutil.net_io_counters().bytes_sent,
                'bytes_recv': psutil.net_io_counters().bytes_recv,
                'packets_sent': psutil.net_io_counters().packets_sent,
                'packets_recv': psutil.net_io_counters().packets_recv
            },
            'connections': len(psutil.net_connections()),
            'processes': len(psutil.pids())
        }
        return stats
    except Exception as e:
        return {'error': str(e)}

def monitor_system():
    """系统监控线程"""
    global system_stats
    
    while True:
        try:
            stats = get_system_stats()
            system_stats = stats
            
            # 实时推送系统状态
            socketio.emit('system_stats', stats)
            
            time.sleep(3)  # 每3秒更新一次
            
        except Exception as e:
            print(f"系统监控错误: {e}")
            time.sleep(5)

# 启动系统监控线程
monitor_thread = threading.Thread(target=monitor_system, daemon=True)
monitor_thread.start()

@app.route('/')
def index():
    """主页面"""
    return render_template('index.html')

@app.route('/api/tasks', methods=['GET'])
def get_tasks():
    """获取所有任务"""
    tasks = []
    for task_id, task in task_manager.tasks.items():
        task_info = task.copy()
        if 'process' in task_info:
            del task_info['process']  # 移除不可序列化的对象
        tasks.append(task_info)
    
    return jsonify(tasks)

@app.route('/api/tasks', methods=['POST'])
def create_task():
    """创建新任务"""
    try:
        config = request.json
        
        # 验证必要参数
        required_fields = ['mode', 'url', 'threads', 'rps']
        for field in required_fields:
            if field not in config:
                return jsonify({'error': f'缺少必要参数: {field}'}), 400
                
        task_id = task_manager.create_task(config)
        
        # 如果配置了立即启动
        if config.get('auto_start', False):
            if task_manager.start_task(task_id):
                return jsonify({
                    'task_id': task_id,
                    'message': '任务创建并启动成功'
                })
            else:
                return jsonify({
                    'task_id': task_id,
                    'message': '任务创建成功，但启动失败'
                })
        
        return jsonify({
            'task_id': task_id,
            'message': '任务创建成功'
        })
        
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/tasks/<task_id>/start', methods=['POST'])
def start_task(task_id):
    """启动任务"""
    try:
        if task_manager.start_task(task_id):
            return jsonify({'message': '任务启动成功'})
        else:
            return jsonify({'error': '任务启动失败'}), 400
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/tasks/<task_id>/stop', methods=['POST'])
def stop_task(task_id):
    """停止任务"""
    try:
        if task_manager.stop_task(task_id):
            return jsonify({'message': '任务停止成功'})
        else:
            return jsonify({'error': '任务停止失败'}), 400
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/tasks/<task_id>/logs')
def get_task_logs(task_id):
    """获取任务日志"""
    if task_id not in task_manager.tasks:
        return jsonify({'error': '任务不存在'}), 404
        
    task = task_manager.tasks[task_id]
    return jsonify(task['logs'])

@app.route('/api/system/stats')
def get_system_stats_api():
    """获取系统状态"""
    return jsonify(system_stats)

@socketio.on('connect')
def handle_connect():
    """WebSocket连接处理"""
    print('客户端已连接')
    emit('system_stats', system_stats)

@socketio.on('disconnect')
def handle_disconnect():
    """WebSocket断开处理"""
    print('客户端已断开')

def get_local_ip():
    """获取本机IP地址"""
    import socket
    try:
        # 连接到一个不存在的地址来获取本机IP
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except Exception:
        return "127.0.0.1"

if __name__ == '__main__':
    # 创建模板目录
    os.makedirs('templates', exist_ok=True)
    
    # 设置日志
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s'
    )
    
    # 获取IP地址
    local_ip = get_local_ip()
    port = 5000
    
    print("=" * 60)
    print("🚀 DDoS压测工具 Web控制面板")
    print("=" * 60)
    print(f"📡 本地访问: http://localhost:{port}")
    print(f"🌐 远程访问: http://{local_ip}:{port}")
    print("=" * 60)
    print("📊 功能特点:")
    print("  • 实时系统监控 (CPU/内存/网络)")
    print("  • 任务管理和调度")
    print("  • 实时日志查看")
    print("  • 性能统计分析")
    print("=" * 60)
    print("⚠️  注意: 仅用于授权的安全测试")
    print("🔴 按 Ctrl+C 停止服务")
    print("=" * 60)
    
    try:
        socketio.run(app, host='0.0.0.0', port=port, debug=False)
    except KeyboardInterrupt:
        print("\n🛑 Web控制面板已停止")
    except Exception as e:
        print(f"\n❌ 启动失败: {e}")
        print("💡 请检查端口是否被占用或依赖是否完整安装")
