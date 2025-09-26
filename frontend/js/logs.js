// 全局变量
let ws = null;
let autoScroll = true;
let currentTaskId = null;
let tasks = [];

// API基础URL
const API_BASE = '/api';

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    initWebSocket();
    loadTasks();
    
    // 从URL参数获取任务ID
    const urlParams = new URLSearchParams(window.location.search);
    const taskId = urlParams.get('task');
    if (taskId) {
        selectTaskById(taskId);
    }
});

// 初始化WebSocket连接
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(protocol + '//' + window.location.host + API_BASE + '/ws');
    
    ws.onopen = function() {
        console.log('WebSocket连接已建立');
    };
    
    ws.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'task_log' && data.task_id === currentTaskId) {
                addLogEntry(data.log);
            } else if (data.type === 'task_stats' && data.task_id === currentTaskId) {
                updateTaskStats(data.stats);
            }
        } catch (error) {
            console.error('WebSocket消息解析失败:', error);
        }
    };
    
    ws.onclose = function() {
        console.log('WebSocket连接已关闭，5秒后重连...');
        setTimeout(initWebSocket, 5000);
    };
}

// 显示加载状态
function showLoading() {
    document.querySelector('.loading').classList.add('show');
}

// 隐藏加载状态
function hideLoading() {
    document.querySelector('.loading').classList.remove('show');
}

// 加载任务列表
async function loadTasks() {
    showLoading();
    try {
        const response = await fetch(API_BASE + '/tasks');
        if (!response.ok) {
            throw new Error('获取任务列表失败');
        }
        tasks = await response.json();
        populateTaskSelect();
        
        // 如果没有URL参数且任务列表不为空，自动选择第一个任务
        const urlParams = new URLSearchParams(window.location.search);
        const taskId = urlParams.get('task');
        if (!taskId && tasks.length > 0) {
            selectTaskById(tasks[0].id);
        }
    } catch (error) {
        console.error('加载任务失败:', error);
        alert('加载任务失败: ' + error.message);
    } finally {
        hideLoading();
    }
}

// 填充任务选择下拉框
function populateTaskSelect() {
    const select = document.getElementById('taskSelect');
    select.innerHTML = '<option value="">请选择任务...</option>';
    
    tasks.forEach(task => {
        const option = document.createElement('option');
        option.value = task.id;
        option.textContent = `${task.name} (${task.target_url}) - ${getStatusText(task.status)}`;
        select.appendChild(option);
    });
}

// 选择任务
function selectTask() {
    const taskId = document.getElementById('taskSelect').value;
    if (taskId) {
        selectTaskById(taskId);
    } else {
        hideTaskDetails();
    }
}

// 根据ID选择任务
function selectTaskById(taskId) {
    const task = tasks.find(t => t.id === taskId);
    if (task) {
        currentTaskId = taskId;
        document.getElementById('taskSelect').value = taskId;
        document.getElementById('taskInfo').textContent = `任务: ${task.name} (${task.target_url})`;
        showTaskDetails();
        loadTaskLogs();
        loadTaskStats();
    }
}

// 显示任务详情
function showTaskDetails() {
    document.getElementById('taskStats').style.display = 'block';
    document.getElementById('logControls').style.display = 'block';
    document.getElementById('logRow').style.display = 'block';
}

// 隐藏任务详情
function hideTaskDetails() {
    currentTaskId = null;
    document.getElementById('taskInfo').textContent = '选择任务查看日志';
    document.getElementById('taskStats').style.display = 'none';
    document.getElementById('logControls').style.display = 'none';
    document.getElementById('logRow').style.display = 'none';
    document.getElementById('logContainer').innerHTML = '';
}

// 刷新任务列表
async function refreshTasks() {
    await loadTasks();
}

// 加载任务日志
async function loadTaskLogs() {
    if (!currentTaskId) return;
    
    showLoading();
    try {
        const response = await fetch(API_BASE + `/tasks/${currentTaskId}/logs`);
        if (!response.ok) {
            throw new Error('获取任务日志失败');
        }
        const logs = await response.json();
        renderLogs(logs);
    } catch (error) {
        console.error('加载日志失败:', error);
        alert('加载日志失败: ' + error.message);
    } finally {
        hideLoading();
    }
}

// 加载任务统计
async function loadTaskStats() {
    if (!currentTaskId) return;
    
    try {
        const response = await fetch(API_BASE + `/tasks/${currentTaskId}/stats`);
        if (!response.ok) {
            throw new Error('获取任务统计失败');
        }
        const stats = await response.json();
        updateTaskStats(stats);
    } catch (error) {
        console.error('加载统计失败:', error);
    }
}

// 渲染日志
function renderLogs(logs) {
    const container = document.getElementById('logContainer');
    container.innerHTML = '';
    
    logs.forEach(log => {
        addLogEntry(log);
    });
}

// 添加日志条目
function addLogEntry(log) {
    const container = document.getElementById('logContainer');
    const div = document.createElement('div');
    div.className = 'log-entry';
    
    // 根据日志级别设置样式
    if (log.includes('ERROR') || log.includes('错误')) {
        div.className += ' log-error';
    } else if (log.includes('WARNING') || log.includes('警告')) {
        div.className += ' log-warning';
    } else if (log.includes('SUCCESS') || log.includes('成功')) {
        div.className += ' log-success';
    } else if (log.includes('DEBUG') || log.includes('调试')) {
        div.className += ' log-debug';
    } else {
        div.className += ' log-info';
    }
    
    div.textContent = log;
    container.appendChild(div);
    
    // 自动滚动到底部
    if (autoScroll) {
        container.scrollTop = container.scrollHeight;
    }
}

// 清空日志
function clearLogs() {
    document.getElementById('logContainer').innerHTML = '';
}

// 切换自动滚动
function toggleAutoScroll() {
    autoScroll = !autoScroll;
    document.getElementById('autoScrollText').textContent = autoScroll ? '自动滚动' : '手动滚动';
}

// 刷新日志
async function refreshLogs() {
    await loadTaskLogs();
}

// 更新任务统计
function updateTaskStats(stats) {
    const container = document.getElementById('taskStats');
    container.innerHTML = `
        <div class="col-md-3">
            <div class="card stats-card">
                <div class="card-body text-center">
                    <h6 class="card-title">总请求数</h6>
                    <h4>${(stats.total_requests || 0).toLocaleString()}</h4>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card stats-card">
                <div class="card-body text-center">
                    <h6 class="card-title">成功请求</h6>
                    <h4>${(stats.successful_requests || 0).toLocaleString()}</h4>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card stats-card">
                <div class="card-body text-center">
                    <h6 class="card-title">失败请求</h6>
                    <h4>${(stats.failed_requests || 0).toLocaleString()}</h4>
                </div>
            </div>
        </div>
        <div class="col-md-3">
            <div class="card stats-card">
                <div class="card-body text-center">
                    <h6 class="card-title">当前RPS</h6>
                    <h4>${(stats.current_rps || 0).toFixed(2)}</h4>
                </div>
            </div>
        </div>
    `;
}

// 获取状态文本
function getStatusText(status) {
    const texts = {
        'pending': '待启动',
        'running': '运行中',
        'completed': '已完成',
        'failed': '失败',
        'stopped': '已停止'
    };
    return texts[status] || status;
}
