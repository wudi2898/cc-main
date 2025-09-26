// 全局变量
let ws = null;
let tasks = [];
let currentTask = null;

// API基础URL
const API_BASE = '/api';

// 初始化应用
document.addEventListener('DOMContentLoaded', function() {
    initWebSocket();
    refreshTasks();
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

// 刷新任务列表
async function refreshTasks() {
    showLoading();
    try {
        const response = await fetch(API_BASE + '/tasks');
        if (!response.ok) {
            throw new Error('获取任务列表失败');
        }
        tasks = await response.json();
        renderTasks();
        updateStats();
    } catch (error) {
        console.error('刷新任务失败:', error);
        alert('刷新任务失败: ' + error.message);
    } finally {
        hideLoading();
    }
}

// 渲染任务列表
function renderTasks() {
    const container = document.getElementById('tasksContainer');
    container.innerHTML = '';

    if (tasks.length === 0) {
        container.innerHTML = `
            <div class="col-12">
                <div class="card">
                    <div class="card-body text-center">
                        <h5 class="card-title">暂无任务</h5>
                        <p class="card-text">点击"创建新任务"按钮开始创建第一个任务</p>
                    </div>
                </div>
            </div>
        `;
        return;
    }

    tasks.forEach(task => {
        const taskCard = createTaskCard(task);
        container.appendChild(taskCard);
    });
}

// 创建任务卡片
function createTaskCard(task) {
    const div = document.createElement('div');
    div.className = 'col-md-6 col-lg-4 mb-3';
    div.innerHTML = `
        <div class="card task-card">
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-start mb-2">
                    <h6 class="card-title">${escapeHtml(task.name)}</h6>
                    <span class="badge status-badge bg-${getStatusColor(task.status)}">${getStatusText(task.status)}</span>
                </div>
                <p class="card-text">
                    <small class="text-muted">目标: ${escapeHtml(task.target_url)}</small><br>
                    <small class="text-muted">模式: ${task.mode.toUpperCase()}</small><br>
                    <small class="text-muted">线程: ${task.threads.toLocaleString()} | RPS: ${task.rps.toLocaleString()}</small>
                </p>
                <div class="d-flex justify-content-between">
                    <div>
                        ${task.status === 'running' ? 
                            `<button class="btn btn-warning btn-sm btn-action" onclick="stopTask('${task.id}')">
                                <i class="bi bi-stop-circle"></i> 停止
                            </button>` :
                            `<button class="btn btn-success btn-sm btn-action" onclick="startTask('${task.id}')">
                                <i class="bi bi-play-circle"></i> 启动
                            </button>`
                        }
                        <button class="btn btn-info btn-sm btn-action" onclick="viewLogs('${task.id}')">
                            <i class="bi bi-journal-text"></i> 日志
                        </button>
                        <button class="btn btn-primary btn-sm btn-action" onclick="editTask('${task.id}')">
                            <i class="bi bi-pencil"></i> 编辑
                        </button>
                        <button class="btn btn-danger btn-sm btn-action" onclick="deleteTask('${task.id}')">
                            <i class="bi bi-trash"></i> 删除
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    return div;
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
    const total = tasks.length;
    const running = tasks.filter(t => t.status === 'running').length;
    const completed = tasks.filter(t => t.status === 'completed').length;
    const failed = tasks.filter(t => t.status === 'failed').length;

    document.getElementById('totalTasks').textContent = total;
    document.getElementById('runningTasks').textContent = running;
    document.getElementById('completedTasks').textContent = completed;
    document.getElementById('failedTasks').textContent = failed;
}

// 创建任务
async function createTask() {
    const formData = {
        name: document.getElementById('taskName').value,
        target_url: document.getElementById('targetURL').value,
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

    if (!formData.name || !formData.target_url) {
        alert('请填写任务名称和目标URL');
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
            throw new Error('创建任务失败');
        }

        bootstrap.Modal.getInstance(document.getElementById('createTaskModal')).hide();
        document.getElementById('createTaskForm').reset();
        refreshTasks();
    } catch (error) {
        console.error('创建任务失败:', error);
        alert('创建任务失败: ' + error.message);
    } finally {
        hideLoading();
    }
}

// 启动任务
async function startTask(taskId) {
    showLoading();
    try {
        const response = await fetch(API_BASE + `/tasks/${taskId}/start`, {
            method: 'POST'
        });
        if (!response.ok) {
            throw new Error('启动任务失败');
        }
        refreshTasks();
    } catch (error) {
        console.error('启动任务失败:', error);
        alert('启动任务失败: ' + error.message);
    } finally {
        hideLoading();
    }
}

// 停止任务
async function stopTask(taskId) {
    showLoading();
    try {
        const response = await fetch(API_BASE + `/tasks/${taskId}/stop`, {
            method: 'POST'
        });
        if (!response.ok) {
            throw new Error('停止任务失败');
        }
        refreshTasks();
    } catch (error) {
        console.error('停止任务失败:', error);
        alert('停止任务失败: ' + error.message);
    } finally {
        hideLoading();
    }
}

// 删除任务
async function deleteTask(taskId) {
    if (confirm('确定要删除这个任务吗？')) {
        showLoading();
        try {
            const response = await fetch(API_BASE + `/tasks/${taskId}`, {
                method: 'DELETE'
            });
            if (!response.ok) {
                throw new Error('删除任务失败');
            }
            refreshTasks();
        } catch (error) {
            console.error('删除任务失败:', error);
            alert('删除任务失败: ' + error.message);
        } finally {
            hideLoading();
        }
    }
}

// 编辑任务
function editTask(taskId) {
    const task = tasks.find(t => t.id === taskId);
    if (task) {
        // 填充编辑表单
        document.getElementById('editTaskId').value = taskId;
        document.getElementById('editTaskForm').innerHTML = document.getElementById('createTaskForm').innerHTML;
        
        // 填充数据
        document.querySelector('#editTaskForm #taskName').value = task.name;
        document.querySelector('#editTaskForm #targetURL').value = task.target_url;
        document.querySelector('#editTaskForm #attackMode').value = task.mode;
        document.querySelector('#editTaskForm #threads').value = task.threads;
        document.querySelector('#editTaskForm #rps').value = task.rps;
        document.querySelector('#editTaskForm #duration').value = task.duration;
        document.querySelector('#editTaskForm #timeout').value = task.timeout;
        document.querySelector('#editTaskForm #cfBypass').checked = task.cf_bypass;
        document.querySelector('#editTaskForm #randomPath').checked = task.random_path;
        document.querySelector('#editTaskForm #randomParams').checked = task.random_params;
        
        // 显示编辑模态框
        new bootstrap.Modal(document.getElementById('editTaskModal')).show();
    }
}

// 更新任务
async function updateTask() {
    const taskId = document.getElementById('editTaskId').value;
    const formData = {
        name: document.querySelector('#editTaskForm #taskName').value,
        target_url: document.querySelector('#editTaskForm #targetURL').value,
        mode: document.querySelector('#editTaskForm #attackMode').value,
        threads: parseInt(document.querySelector('#editTaskForm #threads').value),
        rps: parseInt(document.querySelector('#editTaskForm #rps').value),
        duration: parseInt(document.querySelector('#editTaskForm #duration').value),
        timeout: parseInt(document.querySelector('#editTaskForm #timeout').value),
        cf_bypass: document.querySelector('#editTaskForm #cfBypass').checked,
        random_path: document.querySelector('#editTaskForm #randomPath').checked,
        random_params: document.querySelector('#editTaskForm #randomParams').checked
    };

    if (!formData.name || !formData.target_url) {
        alert('请填写任务名称和目标URL');
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
            throw new Error('更新任务失败');
        }

        bootstrap.Modal.getInstance(document.getElementById('editTaskModal')).hide();
        refreshTasks();
    } catch (error) {
        console.error('更新任务失败:', error);
        alert('更新任务失败: ' + error.message);
    } finally {
        hideLoading();
    }
}

// 查看日志
function viewLogs(taskId) {
    window.open(`logs.html?task=${taskId}`, '_blank');
}

// 更新任务列表中的任务
function updateTaskInList(updatedTask) {
    const index = tasks.findIndex(t => t.id === updatedTask.id);
    if (index !== -1) {
        tasks[index] = updatedTask;
        renderTasks();
        updateStats();
    }
}

// 添加日志条目
function addLogEntry(taskId, logEntry) {
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
