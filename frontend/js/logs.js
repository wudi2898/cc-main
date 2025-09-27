// 全局变量
let ws = null;
let autoScroll = true;
let currentTaskId = null;
let tasks = [];
let currentFilter = 'all';
let searchTerm = '';

// API基础URL
const API_BASE = '/api';

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    // 确保tasks数组初始化
    if (!tasks || !Array.isArray(tasks)) {
        tasks = [];
    }
    
    initWebSocket();
    loadTasks();
    initEventListeners();
    
    // 从URL参数获取任务ID
    const urlParams = new URLSearchParams(window.location.search);
    const taskId = urlParams.get('task');
    if (taskId) {
        selectTaskById(taskId);
    }
});

// 初始化事件监听器
function initEventListeners() {
    // 搜索框
    const searchInput = document.getElementById('searchInput');
    if (searchInput) {
        searchInput.addEventListener('input', function() {
            searchTerm = this.value.toLowerCase();
            filterLogs();
        });
    }
    
    // 过滤器按钮
    const filterBtns = document.querySelectorAll('.filter-btn');
    filterBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            // 移除所有active类
            filterBtns.forEach(b => b.classList.remove('active'));
            // 添加active类到当前按钮
            this.classList.add('active');
            // 设置当前过滤器
            currentFilter = this.dataset.level;
            filterLogs();
        });
    });
}

// 初始化WebSocket连接
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(protocol + '//' + window.location.host + API_BASE + '/ws');
    
    ws.onopen = function() {
        console.log('WebSocket连接已建立');
        showToast('连接成功', 'success');
    };
    
    ws.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'task_update') {
                updateTaskInList(data.task);
            } else if (data.type === 'task_log' && data.task_id === currentTaskId) {
                addLogEntry(data.log);
            } else if (data.type === 'task_stats' && data.task_id === currentTaskId) {
                updateTaskStats(data.stats);
            }
        } catch (error) {
            console.error('WebSocket消息解析失败:', error);
        }
    };
    
    ws.onerror = function(error) {
        console.error('WebSocket连接错误:', error);
        showToast('连接错误', 'error');
    };
    
    ws.onclose = function() {
        console.log('WebSocket连接已关闭，5秒后重连...');
        showToast('连接断开，正在重连...', 'warning');
        setTimeout(initWebSocket, 5000);
    };
}

// 显示加载状态
function showLoading() {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) {
        overlay.classList.add('show');
    }
}

// 隐藏加载状态
function hideLoading() {
    const overlay = document.getElementById('loadingOverlay');
    if (overlay) {
        overlay.classList.remove('show');
    }
}

// 显示Toast消息
function showToast(message, type = 'info') {
    const toastContainer = document.getElementById('toastContainer');
    if (!toastContainer) return;
    
    const toastId = 'toast-' + Date.now();
    
    const toastHtml = `
        <div class="toast" id="${toastId}" role="alert" aria-live="assertive" aria-atomic="true">
            <div class="toast-header bg-${type === 'error' ? 'danger' : type} text-white">
                <i class="bi bi-${getToastIcon(type)} me-2"></i>
                <strong class="me-auto">${getToastTitle(type)}</strong>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="toast"></button>
            </div>
            <div class="toast-body">
                ${message}
            </div>
        </div>
    `;
    
    toastContainer.insertAdjacentHTML('beforeend', toastHtml);
    
    const toastElement = document.getElementById(toastId);
    const toast = new bootstrap.Toast(toastElement, { delay: 3000 });
    toast.show();
    
    // 自动移除
    toastElement.addEventListener('hidden.bs.toast', () => {
        toastElement.remove();
    });
}

// 获取Toast图标
function getToastIcon(type) {
    const icons = {
        'success': 'check-circle-fill',
        'error': 'x-circle-fill',
        'warning': 'exclamation-triangle-fill',
        'info': 'info-circle-fill'
    };
    return icons[type] || 'info-circle-fill';
}

// 获取Toast标题
function getToastTitle(type) {
    const titles = {
        'success': '成功',
        'error': '错误',
        'warning': '警告',
        'info': '信息'
    };
    return titles[type] || '信息';
}

// 加载任务列表
async function loadTasks() {
    showLoading();
    try {
        const response = await fetch(API_BASE + '/tasks');
        if (!response.ok) {
            throw new Error('获取任务列表失败');
        }
        const data = await response.json();
        tasks = Array.isArray(data) ? data : [];
        populateTaskSelect();
        
        // 如果没有URL参数且任务列表不为空，自动选择第一个任务
        const urlParams = new URLSearchParams(window.location.search);
        const taskId = urlParams.get('task');
        if (!taskId && tasks.length > 0) {
            selectTaskById(tasks[0].id);
        }
        
        showToast('任务列表加载成功', 'success');
    } catch (error) {
        console.error('加载任务失败:', error);
        showToast('加载任务失败: ' + error.message, 'error');
    } finally {
        hideLoading();
    }
}

// 填充任务选择下拉框
function populateTaskSelect() {
    const select = document.getElementById('taskSelect');
    if (!select) return;
    
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
    const select = document.getElementById('taskSelect');
    if (!select) return;
    
    const taskId = select.value;
    if (taskId) {
        selectTaskById(taskId);
    } else {
        hideTaskDetails();
    }
}

// 根据ID选择任务
function selectTaskById(taskId) {
    if (!tasks || !Array.isArray(tasks)) {
        showToast('任务列表未加载', 'error');
        return;
    }
    
    const task = tasks.find(t => t.id === taskId);
    if (task) {
        currentTaskId = taskId;
        const select = document.getElementById('taskSelect');
        if (select) {
            select.value = taskId;
        }
        
        const taskInfo = document.getElementById('taskInfo');
        if (taskInfo) {
            taskInfo.textContent = `任务: ${task.name} (${task.target_url})`;
        }
        
        showTaskDetails();
        loadTaskLogs();
        loadTaskStats();
        updateTaskStatus(task.status);
    }
}

// 显示任务详情
function showTaskDetails() {
    const elements = [
        'taskStats', 'logControls', 'logCard', 'logFilters', 'searchBox'
    ];
    
    elements.forEach(id => {
        const element = document.getElementById(id);
        if (element) {
            element.style.display = element.id === 'taskStats' ? 'grid' : 'block';
        }
    });
    
    const emptyState = document.getElementById('emptyState');
    if (emptyState) {
        emptyState.style.display = 'none';
    }
}

// 隐藏任务详情
function hideTaskDetails() {
    currentTaskId = null;
    
    const taskInfo = document.getElementById('taskInfo');
    if (taskInfo) {
        taskInfo.textContent = '选择任务查看日志';
    }
    
    const elements = [
        'taskStats', 'logControls', 'logCard', 'logFilters', 'searchBox'
    ];
    
    elements.forEach(id => {
        const element = document.getElementById(id);
        if (element) {
            element.style.display = 'none';
        }
    });
    
    const logContainer = document.getElementById('logContainer');
    if (logContainer) {
        logContainer.innerHTML = '';
    }
    
    const emptyState = document.getElementById('emptyState');
    if (emptyState) {
        emptyState.style.display = 'block';
    }
}

// 更新任务状态
function updateTaskStatus(status) {
    const statusIndicator = document.getElementById('taskStatus');
    if (!statusIndicator) return;
    
    const statusText = statusIndicator.querySelector('span');
    const statusDot = statusIndicator.querySelector('.status-dot');
    
    if (statusText) {
        statusText.textContent = getStatusText(status);
    }
    
    // 更新状态样式
    statusIndicator.className = `status-indicator status-${status}`;
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
        showToast('日志加载成功', 'success');
    } catch (error) {
        console.error('加载日志失败:', error);
        showToast('加载日志失败: ' + error.message, 'error');
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
    if (!container) return;
    
    container.innerHTML = '';
    
    if (!logs || !Array.isArray(logs)) {
        return;
    }
    
    logs.forEach(log => {
        addLogEntry(log);
    });
}

// 添加日志条目
function addLogEntry(log) {
    const container = document.getElementById('logContainer');
    if (!container) return;
    
    const div = document.createElement('div');
    div.className = 'log-entry';
    
    // 解析日志格式 [时间] 内容
    const logMatch = log.match(/^\[([^\]]+)\]\s*(.*)$/);
    let timestamp = '';
    let message = log;
    
    if (logMatch) {
        timestamp = logMatch[1];
        message = logMatch[2];
    }
    
    // 根据日志级别设置样式
    let level = 'info';
    if (message.includes('ERROR') || message.includes('错误') || message.includes('失败')) {
        level = 'error';
    } else if (message.includes('WARNING') || message.includes('警告')) {
        level = 'warning';
    } else if (message.includes('SUCCESS') || message.includes('成功')) {
        level = 'success';
    } else if (message.includes('DEBUG') || message.includes('调试')) {
        level = 'debug';
    }
    
    div.className += ` log-${level}`;
    
    // 构建日志内容
    div.innerHTML = `
        ${timestamp ? `<span class="log-timestamp">[${timestamp}]</span>` : ''}
        <span class="log-level">${level.toUpperCase()}</span>
        <span class="log-message">${escapeHtml(message)}</span>
    `;
    
    container.appendChild(div);
    
    // 自动滚动到底部
    if (autoScroll) {
        container.scrollTop = container.scrollHeight;
    }
}

// 过滤日志
function filterLogs() {
    const container = document.getElementById('logContainer');
    if (!container) return;
    
    const entries = container.querySelectorAll('.log-entry');
    entries.forEach(entry => {
        let show = true;
        
        // 按级别过滤
        if (currentFilter !== 'all') {
            const level = entry.className.match(/log-(\w+)/);
            if (level && level[1] !== currentFilter) {
                show = false;
            }
        }
        
        // 按搜索词过滤
        if (show && searchTerm) {
            const text = entry.textContent.toLowerCase();
            if (!text.includes(searchTerm)) {
                show = false;
            }
        }
        
        entry.style.display = show ? 'block' : 'none';
    });
}

// 清空日志
function clearLogs() {
    const container = document.getElementById('logContainer');
    if (container) {
        container.innerHTML = '';
        showToast('日志已清空', 'success');
    }
}

// 切换自动滚动
function toggleAutoScroll() {
    autoScroll = !autoScroll;
    const text = document.getElementById('autoScrollText');
    if (text) {
        text.textContent = autoScroll ? '自动滚动' : '手动滚动';
    }
    showToast(autoScroll ? '已开启自动滚动' : '已关闭自动滚动', 'info');
}

// 刷新日志
async function refreshLogs() {
    await loadTaskLogs();
}

// 导出日志
function exportLogs() {
    const container = document.getElementById('logContainer');
    if (!container) {
        showToast('没有日志可导出', 'warning');
        return;
    }
    
    const entries = container.querySelectorAll('.log-entry');
    if (entries.length === 0) {
        showToast('没有日志可导出', 'warning');
        return;
    }
    
    let logText = '';
    entries.forEach(entry => {
        logText += entry.textContent + '\n';
    });
    
    const blob = new Blob([logText], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    
    const link = document.createElement('a');
    link.href = url;
    link.download = `logs_${currentTaskId}_${new Date().toISOString().split('T')[0]}.txt`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    
    showToast('日志导出成功', 'success');
}

// 更新任务统计
function updateTaskStats(stats) {
    const container = document.getElementById('taskStats');
    if (!container) return;
    
    container.innerHTML = `
        <div class="stat-card">
            <div class="stat-value">${(stats.total_requests || 0).toLocaleString()}</div>
            <div class="stat-label">总请求数</div>
        </div>
        <div class="stat-card">
            <div class="stat-value">${(stats.successful_requests || 0).toLocaleString()}</div>
            <div class="stat-label">成功请求</div>
        </div>
        <div class="stat-card">
            <div class="stat-value">${(stats.failed_requests || 0).toLocaleString()}</div>
            <div class="stat-label">失败请求</div>
        </div>
        <div class="stat-card">
            <div class="stat-value">${(stats.current_rps || 0).toFixed(0)}</div>
            <div class="stat-label">当前RPS</div>
        </div>
        <div class="stat-card">
            <div class="stat-value">${(stats.avg_rps || 0).toFixed(0)}</div>
            <div class="stat-label">平均RPS</div>
        </div>
        <div class="stat-card">
            <div class="stat-value">${((stats.successful_requests || 0) / Math.max(stats.total_requests || 1, 1) * 100).toFixed(1)}%</div>
            <div class="stat-label">成功率</div>
        </div>
    `;
}

// 更新任务列表中的任务
function updateTaskInList(updatedTask) {
    if (!tasks || !Array.isArray(tasks)) {
        tasks = [];
    }
    const index = tasks.findIndex(t => t.id === updatedTask.id);
    if (index !== -1) {
        tasks[index] = updatedTask;
        populateTaskSelect();
        
        // 如果当前选中的任务被更新，更新状态
        if (currentTaskId === updatedTask.id) {
            updateTaskStatus(updatedTask.status);
            if (updatedTask.stats) {
                updateTaskStats(updatedTask.stats);
            }
        }
    }
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

// HTML转义
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 页面卸载时清理
window.addEventListener('beforeunload', function() {
    if (ws) {
        ws.close();
    }
});