// 全局变量
let ws = null;
let tasks = [];
let currentTask = null;
let autoRefreshInterval = null;

// API基础URL
const API_BASE = '/api';

// 初始化应用
document.addEventListener('DOMContentLoaded', function() {
    // 确保tasks数组初始化
    if (!tasks || !Array.isArray(tasks)) {
        tasks = [];
    }
    
    initWebSocket();
    refreshTasks();
    startAutoRefresh();
    
    // 添加键盘快捷键
    document.addEventListener('keydown', handleKeyboardShortcuts);
});

// 键盘快捷键
function handleKeyboardShortcuts(e) {
    if (e.ctrlKey || e.metaKey) {
        switch(e.key) {
            case 'r':
                e.preventDefault();
                refreshTasks();
                break;
            case 'n':
                e.preventDefault();
                document.getElementById('createTaskModal').click();
                break;
        }
    }
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
            } else if (data.type === 'task_log') {
                addLogEntry(data.task_id, data.log);
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

// 自动刷新
function startAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }
    autoRefreshInterval = setInterval(refreshTasks, 30000); // 30秒刷新一次
}

// 显示加载状态
function showLoading() {
    document.getElementById('loadingOverlay').classList.add('show');
    document.getElementById('loadingSpinner').classList.add('show');
}

// 隐藏加载状态
function hideLoading() {
    document.getElementById('loadingOverlay').classList.remove('show');
    document.getElementById('loadingSpinner').classList.remove('show');
}

// 显示Toast消息
function showToast(message, type = 'info') {
    const toastContainer = document.getElementById('toastContainer');
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
        'success': 'check-circle',
        'error': 'x-circle',
        'warning': 'exclamation-triangle',
        'info': 'info-circle'
    };
    return icons[type] || 'info-circle';
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

// 刷新任务列表
async function refreshTasks() {
    showLoading();
    try {
        const response = await fetch(API_BASE + '/tasks');
        if (!response.ok) {
            throw new Error('获取任务列表失败');
        }
        const data = await response.json();
        tasks = Array.isArray(data) ? data : [];
        renderTasks();
        updateStats();
        showToast('任务列表已刷新', 'success');
    } catch (error) {
        console.error('刷新任务失败:', error);
        showToast('刷新任务失败: ' + error.message, 'error');
    } finally {
        hideLoading();
    }
}

// 渲染任务列表
function renderTasks() {
    const container = document.getElementById('tasksContainer');
    container.innerHTML = '';

    if (!tasks || tasks.length === 0) {
        container.innerHTML = `
            <div class="col-12">
                <div class="card task-card">
                    <div class="card-body empty-state">
                        <i class="bi bi-inbox"></i>
                        <h5>暂无任务</h5>
                        <p>点击"创建新任务"按钮开始创建第一个任务</p>
                        <button class="btn btn-primary" data-bs-toggle="modal" data-bs-target="#createTaskModal">
                            <i class="bi bi-plus-circle"></i> 创建新任务
                        </button>
                    </div>
                </div>
            </div>
        `;
        return;
    }

    tasks.forEach((task, index) => {
        const taskCard = createTaskCard(task);
        taskCard.style.animationDelay = `${index * 0.1}s`;
        container.appendChild(taskCard);
    });
}

// 创建任务卡片
function createTaskCard(task) {
    const div = document.createElement('div');
    div.className = 'col-md-6 col-lg-4 mb-3 fade-in';
    
    const statusColor = getStatusColor(task.status);
    const statusIcon = getStatusIcon(task.status);
    const progress = calculateProgress(task);
    
    div.innerHTML = `
        <div class="card task-card">
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-start mb-3">
                    <h6 class="card-title mb-0">${escapeHtml(task.name)}</h6>
                    <span class="badge status-badge bg-${statusColor}">
                        <i class="bi bi-${statusIcon}"></i> ${getStatusText(task.status)}
                    </span>
                </div>
                
                <div class="task-info mb-3">
                    <div class="mb-2">
                        <strong>目标:</strong> ${escapeHtml(task.target_url)}
                    </div>
                    <div class="mb-2">
                        <strong>模式:</strong> ${task.mode.toUpperCase()}
                    </div>
                    <div class="mb-2">
                        <strong>配置:</strong> ${task.threads.toLocaleString()} 线程 | ${task.rps.toLocaleString()} RPS
                    </div>
                    <div class="mb-2">
                        <strong>创建时间:</strong> ${formatDateTime(task.created_at)}
                    </div>
                    ${task.started_at ? `<div class="mb-2"><strong>开始时间:</strong> ${formatDateTime(task.started_at)}</div>` : ''}
                </div>
                
                ${task.stats ? `
                    <div class="row mb-3">
                        <div class="col-6">
                            <small class="text-muted">总请求</small>
                            <div class="fw-bold">${(task.stats.total_requests || 0).toLocaleString()}</div>
                        </div>
                        <div class="col-6">
                            <small class="text-muted">成功率</small>
                            <div class="fw-bold text-${task.stats.successful_requests > 0 ? 'success' : 'danger'}">
                                ${calculateSuccessRate(task.stats)}%
                            </div>
                        </div>
                    </div>
                ` : ''}
                
                <div class="d-flex justify-content-between align-items-center">
                    <div class="btn-group" role="group">
                        ${task.status === 'running' ? 
                            `<button class="btn btn-warning btn-sm btn-action" onclick="stopTask('${task.id}')" title="停止任务">
                                <i class="bi bi-stop-circle"></i>
                            </button>` :
                            `<button class="btn btn-success btn-sm btn-action" onclick="startTask('${task.id}')" title="启动任务">
                                <i class="bi bi-play-circle"></i>
                            </button>`
                        }
                        <button class="btn btn-info btn-sm btn-action" onclick="viewLogs('${task.id}')" title="查看日志">
                            <i class="bi bi-journal-text"></i>
                        </button>
                        <button class="btn btn-primary btn-sm btn-action" onclick="editTask('${task.id}')" title="编辑任务">
                            <i class="bi bi-pencil"></i>
                        </button>
                        <button class="btn btn-danger btn-sm btn-action" onclick="deleteTask('${task.id}')" title="删除任务">
                            <i class="bi bi-trash"></i>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    return div;
}

// 计算成功率
function calculateSuccessRate(stats) {
    if (!stats || !stats.total_requests || stats.total_requests === 0) {
        return 0;
    }
    return ((stats.successful_requests / stats.total_requests) * 100).toFixed(1);
}

// 计算进度
function calculateProgress(task) {
    if (!task.started_at || !task.duration) {
        return 0;
    }
    const startTime = new Date(task.started_at);
    const now = new Date();
    const elapsed = (now - startTime) / 1000;
    const progress = Math.min((elapsed / task.duration) * 100, 100);
    return Math.max(0, progress);
}

// 格式化日期时间
function formatDateTime(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

// 获取状态颜色
function getStatusColor(status) {
    const colors = {
        'pending': 'secondary',
        'running': 'success',
        'completed': 'primary',
        'failed': 'danger',
        'stopped': 'warning'
    };
    return colors[status] || 'secondary';
}

// 获取状态图标
function getStatusIcon(status) {
    const icons = {
        'pending': 'clock',
        'running': 'play-circle',
        'completed': 'check-circle',
        'failed': 'x-circle',
        'stopped': 'pause-circle'
    };
    return icons[status] || 'question-circle';
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

// 更新统计
function updateStats() {
    if (!tasks) {
        tasks = [];
    }
    const total = tasks.length;
    const running = tasks.filter(t => t.status === 'running').length;
    const completed = tasks.filter(t => t.status === 'completed').length;
    const failed = tasks.filter(t => t.status === 'failed').length;

    // 添加动画效果
    animateNumber('totalTasks', total);
    animateNumber('runningTasks', running);
    animateNumber('completedTasks', completed);
    animateNumber('failedTasks', failed);
}

// 数字动画
function animateNumber(elementId, targetValue) {
    const element = document.getElementById(elementId);
    const currentValue = parseInt(element.textContent) || 0;
    
    if (currentValue === targetValue) return;
    
    const increment = (targetValue - currentValue) / 20;
    let current = currentValue;
    
    const timer = setInterval(() => {
        current += increment;
        if ((increment > 0 && current >= targetValue) || (increment < 0 && current <= targetValue)) {
            current = targetValue;
            clearInterval(timer);
        }
        element.textContent = Math.round(current);
    }, 50);
}

// 创建任务
async function createTask() {
    const formData = {
        name: document.getElementById('taskName').value.trim(),
        target_url: document.getElementById('targetURL').value.trim(),
        mode: document.getElementById('attackMode').value,
        threads: parseInt(document.getElementById('threads').value),
        rps: parseInt(document.getElementById('rps').value),
        duration: parseInt(document.getElementById('duration').value),
        timeout: parseInt(document.getElementById('timeout').value),
        cf_bypass: document.getElementById('cfBypass').checked,
        random_path: document.getElementById('randomPath').checked,
        random_params: document.getElementById('randomParams').checked,
        status: document.getElementById('status').value
    };

    // 验证表单
    if (!formData.name) {
        showToast('请输入任务名称', 'error');
        return;
    }
    if (!formData.target_url) {
        showToast('请输入目标URL', 'error');
        return;
    }
    if (formData.threads <= 0 || formData.rps <= 0) {
        showToast('线程数和RPS必须大于0', 'error');
        return;
    }

    showLoading();
    try {
        const response = await fetch(API_BASE + '/tasks', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || '创建任务失败');
        }

        const newTask = await response.json();
        bootstrap.Modal.getInstance(document.getElementById('createTaskModal')).hide();
        document.getElementById('createTaskForm').reset();
        
        showToast(`任务 "${newTask.name}" 创建成功`, 'success');
        refreshTasks();
    } catch (error) {
        console.error('创建任务失败:', error);
        showToast('创建任务失败: ' + error.message, 'error');
    } finally {
        hideLoading();
    }
}

// 启动任务
async function startTask(taskId) {
    if (!tasks || !Array.isArray(tasks)) {
        showToast('任务列表未加载', 'error');
        return;
    }
    const task = tasks.find(t => t.id === taskId);
    if (!task) {
        showToast('任务不存在', 'error');
        return;
    }

    showLoading();
    try {
        const response = await fetch(API_BASE + `/tasks/${taskId}/start`, {
            method: 'POST'
        });
        if (!response.ok) {
            throw new Error('启动任务失败');
        }
        showToast(`任务 "${task.name}" 启动成功`, 'success');
        refreshTasks();
    } catch (error) {
        console.error('启动任务失败:', error);
        showToast('启动任务失败: ' + error.message, 'error');
    } finally {
        hideLoading();
    }
}

// 停止任务
async function stopTask(taskId) {
    if (!tasks || !Array.isArray(tasks)) {
        showToast('任务列表未加载', 'error');
        return;
    }
    const task = tasks.find(t => t.id === taskId);
    if (!task) {
        showToast('任务不存在', 'error');
        return;
    }

    if (!confirm(`确定要停止任务 "${task.name}" 吗？`)) {
        return;
    }

    showLoading();
    try {
        const response = await fetch(API_BASE + `/tasks/${taskId}/stop`, {
            method: 'POST'
        });
        if (!response.ok) {
            throw new Error('停止任务失败');
        }
        showToast(`任务 "${task.name}" 已停止`, 'success');
        refreshTasks();
    } catch (error) {
        console.error('停止任务失败:', error);
        showToast('停止任务失败: ' + error.message, 'error');
    } finally {
        hideLoading();
    }
}

// 删除任务
async function deleteTask(taskId) {
    if (!tasks || !Array.isArray(tasks)) {
        showToast('任务列表未加载', 'error');
        return;
    }
    const task = tasks.find(t => t.id === taskId);
    if (!task) {
        showToast('任务不存在', 'error');
        return;
    }

    if (!confirm(`确定要删除任务 "${task.name}" 吗？此操作不可撤销。`)) {
        return;
    }

    showLoading();
    try {
        const response = await fetch(API_BASE + `/tasks/${taskId}`, {
            method: 'DELETE'
        });
        if (!response.ok) {
            throw new Error('删除任务失败');
        }
        showToast(`任务 "${task.name}" 已删除`, 'success');
        refreshTasks();
    } catch (error) {
        console.error('删除任务失败:', error);
        showToast('删除任务失败: ' + error.message, 'error');
    } finally {
        hideLoading();
    }
}

// 编辑任务
function editTask(taskId) {
    if (!tasks || !Array.isArray(tasks)) {
        showToast('任务列表未加载', 'error');
        return;
    }
    const task = tasks.find(t => t.id === taskId);
    if (!task) {
        showToast('任务不存在', 'error');
        return;
    }
    
    // 填充编辑表单
    document.getElementById('editTaskId').value = taskId;
    document.getElementById('editTaskForm').innerHTML = document.getElementById('createTaskForm').innerHTML;
    
    // 填充数据
    const form = document.getElementById('editTaskForm');
    form.querySelector('#taskName').value = task.name;
    form.querySelector('#targetURL').value = task.target_url;
    form.querySelector('#attackMode').value = task.mode;
    form.querySelector('#threads').value = task.threads;
    form.querySelector('#rps').value = task.rps;
    form.querySelector('#duration').value = task.duration;
    form.querySelector('#timeout').value = task.timeout;
    form.querySelector('#cfBypass').checked = task.cf_bypass;
    form.querySelector('#randomPath').checked = task.random_path;
    form.querySelector('#randomParams').checked = task.random_params;
    
    // 显示编辑模态框
    new bootstrap.Modal(document.getElementById('editTaskModal')).show();
}

// 更新任务
async function updateTask() {
    const taskId = document.getElementById('editTaskId').value;
    const form = document.getElementById('editTaskForm');
    
    const formData = {
        name: form.querySelector('#taskName').value.trim(),
        target_url: form.querySelector('#targetURL').value.trim(),
        mode: form.querySelector('#attackMode').value,
        threads: parseInt(form.querySelector('#threads').value),
        rps: parseInt(form.querySelector('#rps').value),
        duration: parseInt(form.querySelector('#duration').value),
        timeout: parseInt(form.querySelector('#timeout').value),
        cf_bypass: form.querySelector('#cfBypass').checked,
        random_path: form.querySelector('#randomPath').checked,
        random_params: form.querySelector('#randomParams').checked
    };

    // 验证表单
    if (!formData.name) {
        showToast('请输入任务名称', 'error');
        return;
    }
    if (!formData.target_url) {
        showToast('请输入目标URL', 'error');
        return;
    }
    if (formData.threads <= 0 || formData.rps <= 0) {
        showToast('线程数和RPS必须大于0', 'error');
        return;
    }

    showLoading();
    try {
        const response = await fetch(API_BASE + `/tasks/${taskId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || '更新任务失败');
        }

        bootstrap.Modal.getInstance(document.getElementById('editTaskModal')).hide();
        showToast('任务更新成功', 'success');
        refreshTasks();
    } catch (error) {
        console.error('更新任务失败:', error);
        showToast('更新任务失败: ' + error.message, 'error');
    } finally {
        hideLoading();
    }
}

// 查看日志
function viewLogs(taskId) {
    window.open(`logs.html?task=${taskId}`, '_blank');
}

// 导出任务
function exportTasks() {
    if (!tasks || !Array.isArray(tasks) || tasks.length === 0) {
        showToast('没有任务可导出', 'warning');
        return;
    }
    
    const dataStr = JSON.stringify(tasks, null, 2);
    const dataBlob = new Blob([dataStr], {type: 'application/json'});
    const url = URL.createObjectURL(dataBlob);
    
    const link = document.createElement('a');
    link.href = url;
    link.download = `tasks_${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    
    showToast('任务列表已导出', 'success');
}

// 更新任务列表中的任务
function updateTaskInList(updatedTask) {
    if (!tasks || !Array.isArray(tasks)) {
        tasks = [];
    }
    const index = tasks.findIndex(t => t.id === updatedTask.id);
    if (index !== -1) {
        tasks[index] = updatedTask;
        renderTasks();
        updateStats();
    }
}

// 添加日志条目
function addLogEntry(taskId, logEntry) {
    if (!tasks || !Array.isArray(tasks)) {
        return;
    }
    const task = tasks.find(t => t.id === taskId);
    if (task) {
        if (!task.logs) task.logs = [];
        task.logs.push(logEntry);
        if (task.logs.length > 100) {
            task.logs = task.logs.slice(-100); // 只保留最近100条日志
        }
    }
}

// HTML转义
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 页面卸载时清理
window.addEventListener('beforeunload', function() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }
    if (ws) {
        ws.close();
    }
});