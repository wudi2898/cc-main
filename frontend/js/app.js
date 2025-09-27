// 全局变量
let eventSource = null;
let tasks = [];

// 图表相关
let cpuChart, memoryChart, trafficChart, goroutinesChart, networkRxChart, networkTxChart;

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
    
    // 记录启动时间
    window.startTime = Date.now();
    
    initCharts();
    initSSE();
    refreshTasks();
    startAutoRefresh();
    
    // 添加键盘快捷键
    document.addEventListener('keydown', handleKeyboardShortcuts);
});

// 初始化图表
function initCharts() {
    // CPU使用率图表 - 实时折线图
    const cpuCtx = document.getElementById('cpuChart').getContext('2d');
    cpuChart = new Chart(cpuCtx, {
        type: 'line',
        data: {
            labels: ['当前'],
            datasets: [{
                label: 'CPU使用率 (%)',
                data: [0],
                borderColor: '#28a745',
                backgroundColor: 'rgba(40, 167, 69, 0.2)',
                borderWidth: 3,
                fill: true,
                tension: 0.4,
                pointRadius: 6,
                pointHoverRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 100,
                    ticks: {
                        callback: function(value) {
                            return value + '%';
                        }
                    }
                },
                x: {
                    display: false
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });

    // 内存使用率图表 - 实时折线图
    const memoryCtx = document.getElementById('memoryChart').getContext('2d');
    memoryChart = new Chart(memoryCtx, {
        type: 'line',
        data: {
            labels: ['当前'],
            datasets: [{
                label: '内存使用率 (%)',
                data: [0],
                borderColor: '#17a2b8',
                backgroundColor: 'rgba(23, 162, 184, 0.2)',
                borderWidth: 3,
                fill: true,
                tension: 0.4,
                pointRadius: 6,
                pointHoverRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 100,
                    ticks: {
                        callback: function(value) {
                            return value + '%';
                        }
                    }
                },
                x: {
                    display: false
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });

    // 实时流量图表 - 折线图
    const trafficCtx = document.getElementById('trafficChart').getContext('2d');
    trafficChart = new Chart(trafficCtx, {
        type: 'line',
        data: {
            labels: ['当前'],
            datasets: [{
                label: '总请求数',
                data: [0],
                borderColor: '#ffc107',
                backgroundColor: 'rgba(255, 193, 7, 0.2)',
                borderWidth: 3,
                fill: true,
                tension: 0.4,
                pointRadius: 6,
                pointHoverRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        callback: function(value) {
                            return value.toLocaleString();
                        }
                    }
                },
                x: {
                    display: false
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });

    // Goroutines数量图表 - 折线图
    const goroutinesCtx = document.getElementById('goroutinesChart').getContext('2d');
    goroutinesChart = new Chart(goroutinesCtx, {
        type: 'line',
        data: {
            labels: ['当前'],
            datasets: [{
                label: 'Goroutines数量',
                data: [0],
                borderColor: '#6f42c1',
                backgroundColor: 'rgba(111, 66, 193, 0.2)',
                borderWidth: 3,
                fill: true,
                tension: 0.4,
                pointRadius: 6,
                pointHoverRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        callback: function(value) {
                            return value.toLocaleString();
                        }
                    }
                },
                x: {
                    display: false
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });

    // 网络接收速度图表 - 折线图
    const networkRxCtx = document.getElementById('networkRxChart').getContext('2d');
    networkRxChart = new Chart(networkRxCtx, {
        type: 'line',
        data: {
            labels: ['当前'],
            datasets: [{
                label: '网络接收速度 (MB/s)',
                data: [0],
                borderColor: '#dc3545',
                backgroundColor: 'rgba(220, 53, 69, 0.2)',
                borderWidth: 3,
                fill: true,
                tension: 0.4,
                pointRadius: 6,
                pointHoverRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        callback: function(value) {
                            return value.toFixed(2) + ' MB/s';
                        }
                    }
                },
                x: {
                    display: false
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });

    // 网络发送速度图表 - 折线图
    const networkTxCtx = document.getElementById('networkTxChart').getContext('2d');
    networkTxChart = new Chart(networkTxCtx, {
        type: 'line',
        data: {
            labels: ['当前'],
            datasets: [{
                label: '网络发送速度 (MB/s)',
                data: [0],
                borderColor: '#fd7e14',
                backgroundColor: 'rgba(253, 126, 20, 0.2)',
                borderWidth: 3,
                fill: true,
                tension: 0.4,
                pointRadius: 6,
                pointHoverRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        callback: function(value) {
                            return value.toFixed(2) + ' MB/s';
                        }
                    }
                },
                x: {
                    display: false
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });
}

// 更新图表数据 - 只显示当前实时数据
function updateCharts(serverStats, totalRequests) {
    // 更新CPU图表 - 折线图显示当前值
    if (cpuChart) {
        cpuChart.data.datasets[0].data = [serverStats.cpu_usage];
        cpuChart.update('none');
    }
    
    // 更新内存图表 - 折线图显示当前值
    if (memoryChart) {
        memoryChart.data.datasets[0].data = [serverStats.memory_usage];
        memoryChart.update('none');
    }
    
    // 更新流量图表 - 折线图显示当前值
    if (trafficChart) {
        trafficChart.data.datasets[0].data = [totalRequests];
        trafficChart.update('none');
    }
    
    // 更新Goroutines图表 - 折线图显示当前值
    if (goroutinesChart) {
        goroutinesChart.data.datasets[0].data = [serverStats.goroutines];
        goroutinesChart.update('none');
    }
    
    // 更新网络接收速度图表 - 折线图显示当前值
    if (networkRxChart) {
        networkRxChart.data.datasets[0].data = [serverStats.network_rx];
        networkRxChart.update('none');
    }
    
    // 更新网络发送速度图表 - 折线图显示当前值
    if (networkTxChart) {
        networkTxChart.data.datasets[0].data = [serverStats.network_tx];
        networkTxChart.update('none');
    }
}

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

// 初始化SSE连接
function initSSE() {
    eventSource = new EventSource(API_BASE + '/events');
    
    eventSource.onopen = function() {
        // 移除SSE连接日志
    };
    
    eventSource.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'task_update') {
                updateTaskInList(data.task);
            } else if (data.type === 'task_log') {
                addLogEntry(data.task_id, data.log);
            } else if (data.type === 'heartbeat') {
                // 心跳消息，保持连接活跃
                // 减少日志输出
            }
        } catch (error) {
            // 移除SSE消息解析错误日志
        }
    };
    
    eventSource.onerror = function(error) {
        // 移除SSE错误日志
        // 5秒后重连
        setTimeout(initSSE, 5000);
    };
}

// 自动刷新
function startAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }
    autoRefreshInterval = setInterval(() => {
        refreshTasks();
        updateServerStats(); // 同时更新服务器统计和图表
    }, 2000); // 2秒刷新一次，提高实时性
}

// 显示加载状态
function showLoading() {
    document.getElementById('loadingOverlay').classList.add('show');
}

// 隐藏加载状态
function hideLoading() {
    document.getElementById('loadingOverlay').classList.remove('show');
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

// 刷新任务列表
async function refreshTasks() {
    try {
        const response = await fetch(API_BASE + '/tasks');
        if (!response.ok) {
            throw new Error('获取任务列表失败');
        }
        const data = await response.json();
        tasks = Array.isArray(data) ? data : [];
        renderTasks();
        updateStats();
        // 减少toast提示，避免频繁弹窗
    } catch (error) {
        // 移除刷新任务失败日志
        showToast('刷新任务失败: ' + error.message, 'error');
    }
}

// 渲染任务列表
function renderTasks() {
    const container = document.getElementById('tasksContainer');
    
    if (!tasks || tasks.length === 0) {
        if (container.children.length === 0 || !container.querySelector('.empty-state')) {
            container.innerHTML = `
                <div class="col-12">
                    <div class="empty-state">
                        <div class="empty-state-icon">
                            <i class="bi bi-inbox"></i>
                        </div>
                        <div class="empty-state-title">暂无任务</div>
                        <div class="empty-state-text">点击"创建新任务"按钮开始创建第一个任务</div>
                        <button class="btn btn-primary" data-bs-toggle="modal" data-bs-target="#createTaskModal">
                            <i class="bi bi-plus-circle-fill"></i> 创建新任务
                        </button>
                    </div>
                </div>
            `;
        }
        return;
    }

    // 按创建时间排序任务，确保顺序稳定
    const sortedTasks = [...tasks].sort((a, b) => {
        const timeA = new Date(a.created_at || 0).getTime();
        const timeB = new Date(b.created_at || 0).getTime();
        return timeA - timeB; // 升序排列，最早的在前面
    });

    // 检查是否需要更新
    const existingCards = container.querySelectorAll('.task-card');
    if (existingCards.length === sortedTasks.length) {
        // 更新现有卡片而不是重新创建
        sortedTasks.forEach((task, index) => {
            const existingCard = existingCards[index];
            if (existingCard) {
                updateTaskCard(existingCard, task);
            }
        });
        return;
    }

    // 只有在任务数量变化时才重新渲染
    container.innerHTML = '';
    sortedTasks.forEach((task, index) => {
        const taskCard = createTaskCard(task);
        taskCard.style.animationDelay = `${index * 0.1}s`;
        container.appendChild(taskCard);
    });
}

// 创建任务卡片
function createTaskCard(task) {
    const div = document.createElement('div');
    div.className = 'fade-in-up';
    
    const statusClass = getStatusClass(task.status);
    const statusText = getStatusText(task.status);
    const statusIcon = getStatusIcon(task.status);
    
    div.innerHTML = `
        <div class="task-card">
            <div class="task-header">
                <div>
                    <div class="task-title">${escapeHtml(task.name)}</div>
                    <span class="task-status ${statusClass}">
                        <i class="bi bi-${statusIcon}"></i> ${statusText}
                    </span>
                </div>
                <button class="btn btn-sm btn-outline-secondary" onclick="deleteTask('${task.id}')" title="删除任务" style="padding: 0.25rem 0.5rem; border: none; background: none; color: #6c757d;">
                    <i class="bi bi-x-lg"></i>
                </button>
            </div>
            
            <div class="task-info">
                <div class="task-info-item">
                    <span class="task-info-label">目标:</span>
                    <span class="task-info-value">${escapeHtml(task.target_url)}</span>
                </div>
                <div class="task-info-item">
                    <span class="task-info-label">模式:</span>
                    <span class="task-info-value">${task.mode.toUpperCase()}</span>
                </div>
                <div class="task-info-item">
                    <span class="task-info-label">配置:</span>
                    <span class="task-info-value">${task.threads.toLocaleString()} 线程 | ${task.rps.toLocaleString()} RPS</span>
                </div>
                <div class="task-info-item">
                    <span class="task-info-label">创建时间:</span>
                    <span class="task-info-value">${formatDateTime(task.created_at)}</span>
                </div>
                ${task.started_at ? `
                    <div class="task-info-item">
                        <span class="task-info-label">开始时间:</span>
                        <span class="task-info-value">${formatDateTime(task.started_at)}</span>
                    </div>
                ` : ''}
            </div>
            
            ${task.stats ? `
                <div class="task-stats">
                    <div class="task-stats-grid">
                        <div class="task-stat">
                            <div class="task-stat-value">${(task.stats.total_requests || 0).toLocaleString()}</div>
                            <div class="task-stat-label">总请求</div>
                        </div>
                        <div class="task-stat">
                            <div class="task-stat-value">${(task.stats.current_rps || 0).toFixed(0)}</div>
                            <div class="task-stat-label">当前RPS</div>
                        </div>
                        <div class="task-stat">
                            <div class="task-stat-value">${(task.stats.avg_rps || 0).toFixed(0)}</div>
                            <div class="task-stat-label">平均RPS</div>
                        </div>
                    </div>
                </div>
            ` : ''}
            
            <div class="task-actions">
                ${task.status === 'running' ? 
                    `<button class="btn btn-warning btn-sm" onclick="stopTask('${task.id}')" title="停止任务">
                        <i class="bi bi-stop-circle-fill"></i> 停止
                    </button>` :
                    `<button class="btn btn-success btn-sm" onclick="startTask('${task.id}')" title="启动任务">
                        <i class="bi bi-play-circle-fill"></i> 启动
                    </button>`
                }
                <button class="btn btn-info btn-sm" onclick="showLogsModal('${task.id}')" title="查看日志">
                    <i class="bi bi-journal-text"></i> 日志
                </button>
                <button class="btn btn-primary btn-sm" onclick="editTask('${task.id}')" title="编辑任务">
                    <i class="bi bi-pencil-fill"></i> 编辑
                </button>
                <button class="btn btn-danger btn-sm" onclick="deleteTask('${task.id}')" title="删除任务">
                    <i class="bi bi-trash-fill"></i> 删除
                </button>
            </div>
        </div>
    `;
    return div;
}

// 更新任务卡片
function updateTaskCard(cardElement, task) {
    const statusClass = getStatusClass(task.status);
    const statusText = getStatusText(task.status);
    const statusIcon = getStatusIcon(task.status);
    
    // 更新状态
    const statusElement = cardElement.querySelector('.task-status');
    if (statusElement) {
        statusElement.className = `task-status ${statusClass}`;
        statusElement.innerHTML = `<i class="bi bi-${statusIcon}"></i> ${statusText}`;
    }
    
    // 更新统计信息
    const statsContainer = cardElement.querySelector('.task-stats-grid');
    if (statsContainer && task.stats) {
        statsContainer.innerHTML = `
            <div class="task-stat">
                <div class="task-stat-value">${(task.stats.total_requests || 0).toLocaleString()}</div>
                <div class="task-stat-label">总请求</div>
            </div>
            <div class="task-stat">
                <div class="task-stat-value">${(task.stats.current_rps || 0).toFixed(0)}</div>
                <div class="task-stat-label">当前RPS</div>
            </div>
            <div class="task-stat">
                <div class="task-stat-value">${(task.stats.avg_rps || 0).toFixed(0)}</div>
                <div class="task-stat-label">平均RPS</div>
            </div>
        `;
    }
    
    // 更新操作按钮
    const actionsContainer = cardElement.querySelector('.task-actions');
    if (actionsContainer) {
        actionsContainer.innerHTML = `
            ${task.status === 'running' ? 
                `<button class="btn btn-warning btn-sm" onclick="stopTask('${task.id}')" title="停止任务">
                    <i class="bi bi-stop-circle-fill"></i> 停止
                </button>` :
                `<button class="btn btn-success btn-sm" onclick="startTask('${task.id}')" title="启动任务">
                    <i class="bi bi-play-circle-fill"></i> 启动
                </button>`
            }
            <button class="btn btn-info btn-sm" onclick="showLogsModal('${task.id}')" title="查看日志">
                <i class="bi bi-journal-text"></i> 日志
            </button>
            <button class="btn btn-primary btn-sm" onclick="editTask('${task.id}')" title="编辑任务">
                <i class="bi bi-pencil-fill"></i> 编辑
            </button>
            <button class="btn btn-danger btn-sm" onclick="deleteTask('${task.id}')" title="删除任务">
                <i class="bi bi-trash-fill"></i> 删除
            </button>
        `;
    }
}

// 获取状态样式类
function getStatusClass(status) {
    return `status-${status}`;
}

// 获取状态图标
function getStatusIcon(status) {
    const icons = {
        'pending': 'clock-fill',
        'running': 'play-circle-fill',
        'completed': 'check-circle-fill',
        'failed': 'x-circle-fill',
        'stopped': 'pause-circle-fill'
    };
    return icons[status] || 'question-circle-fill';
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

    // 添加动画效果
    animateNumber('totalTasks', total);
    animateNumber('runningTasks', running);
    
    // 更新系统统计
    updateSystemStats();
}

// 更新系统统计
function updateSystemStats() {
    if (!tasks) {
        tasks = [];
    }
    
    // 计算系统总统计
    let totalRequests = 0;
    let totalSuccessful = 0;
    let totalFailed = 0;
    let currentRPS = 0;
    let avgRPS = 0;
    let activeTasks = 0;
    
    tasks.forEach(task => {
        if (task.stats) {
            totalRequests += task.stats.total_requests || 0;
            totalSuccessful += task.stats.successful_requests || 0;
            totalFailed += task.stats.failed_requests || 0;
            currentRPS += task.stats.current_rps || 0;
            avgRPS += task.stats.avg_rps || 0;
        }
        if (task.status === 'running') {
            activeTasks++;
        }
    });
    
    
    // 更新系统统计显示
    animateNumber('totalRequests', totalRequests);
    animateNumber('currentRPS', Math.round(currentRPS));
    animateNumber('avgRPS', Math.round(avgRPS));
    animateNumber('activeTasks', activeTasks);
    
    
    
    // 更新服务器性能统计
    updateServerStats();
}

// 更新服务器性能统计
async function updateServerStats() {
    try {
        const [serverResponse, trafficResponse] = await Promise.all([
            fetch(API_BASE + '/server-stats'),
            fetch(API_BASE + '/traffic-stats')
        ]);
        
        if (serverResponse.ok) {
            const stats = await serverResponse.json();
            
            // 更新服务器性能显示
            const cpuElement = document.getElementById('serverCPU');
            if (cpuElement) {
                cpuElement.textContent = stats.cpu_usage.toFixed(1) + '%';
            }
            
            const memoryElement = document.getElementById('serverMemory');
            if (memoryElement) {
                memoryElement.textContent = stats.memory_usage.toFixed(1) + '%';
            }
            
            const goroutinesElement = document.getElementById('serverGoroutines');
            if (goroutinesElement) {
                goroutinesElement.textContent = stats.goroutines;
            }
            
            const uptimeElement = document.getElementById('serverUptime');
            if (uptimeElement) {
                const hours = Math.floor(stats.uptime / 3600);
                const minutes = Math.floor((stats.uptime % 3600) / 60);
                uptimeElement.textContent = `${hours}h ${minutes}m`;
            }
            
            // 更新图表数据
            if (trafficResponse.ok) {
                const trafficStats = await trafficResponse.json();
                updateCharts(stats, trafficStats.total_requests);
            }
        }
    } catch (error) {
        // 移除获取服务器统计失败日志
    }
}

// 数字动画
function animateNumber(elementId, targetValue) {
    const element = document.getElementById(elementId);
    if (!element) return;
    
    const currentValue = parseInt(element.textContent) || 0;
    
    if (currentValue === targetValue) return;
    
    // 简化动画，直接更新数值，避免频繁的setInterval
    element.textContent = targetValue;
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
        schedule: document.getElementById('schedule').checked,
        schedule_interval: parseInt(document.getElementById('scheduleInterval').value),
        schedule_duration: parseInt(document.getElementById('scheduleDuration').value),
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
    form.querySelector('#editSchedule').checked = task.schedule;
    form.querySelector('#editScheduleInterval').value = task.schedule_interval;
    form.querySelector('#editScheduleDuration').value = task.schedule_duration;
    
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
        random_params: form.querySelector('#randomParams').checked,
        schedule: form.querySelector('#editSchedule').checked,
        schedule_interval: parseInt(form.querySelector('#editScheduleInterval').value),
        schedule_duration: parseInt(form.querySelector('#editScheduleDuration').value)
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

// 显示日志弹窗
function showLogsModal(taskId) {
    const task = tasks.find(t => t.id === taskId);
    if (!task) {
        showToast('任务不存在', 'error');
        return;
    }
    
    // 设置当前任务信息
    document.getElementById('logsTaskName').textContent = task.name;
    document.getElementById('logsTaskUrl').textContent = task.target_url;
    document.getElementById('logsTaskStatus').textContent = getStatusText(task.status);
    document.getElementById('logsTaskStatus').className = `badge bg-${getStatusClass(task.status)}`;
    
    // 清空日志容器
    const logContainer = document.getElementById('logsContainer');
    logContainer.innerHTML = '';
    
    // 设置当前任务ID
    window.currentLogsTaskId = taskId;
    document.getElementById('logsModal').setAttribute('data-task-id', taskId);
    
    // 显示弹窗
    new bootstrap.Modal(document.getElementById('logsModal')).show();
    
    // 加载日志
    loadTaskLogsForModal(taskId);
}

// 为弹窗加载任务日志
async function loadTaskLogsForModal(taskId) {
    try {
        const response = await fetch(API_BASE + `/tasks/${taskId}/logs`);
        if (!response.ok) {
            throw new Error('获取任务日志失败');
        }
        const logs = await response.json();
        renderLogsInModal(logs);
    } catch (error) {
        console.error('加载日志失败:', error);
        showToast('加载日志失败: ' + error.message, 'error');
    }
}

// 在弹窗中渲染日志
function renderLogsInModal(logs) {
    const container = document.getElementById('logsContainer');
    if (!container) return;
    
    container.innerHTML = '';
    
    if (!logs || !Array.isArray(logs)) {
        return;
    }
    
    logs.forEach(log => {
        addLogEntryToModal(log);
    });
}

// 向弹窗添加日志条目
function addLogEntryToModal(log) {
    const container = document.getElementById('logsContainer');
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
    container.scrollTop = container.scrollHeight;
}

// 获取状态样式类
function getStatusClass(status) {
    const classes = {
        'pending': 'secondary',
        'running': 'success',
        'completed': 'primary',
        'failed': 'danger',
        'stopped': 'warning'
    };
    return classes[status] || 'secondary';
}

// 刷新弹窗中的日志
function refreshLogsInModal() {
    const modal = document.getElementById('logsModal');
    if (modal && modal.classList.contains('show')) {
        // 获取当前任务ID（从data属性中获取）
        const currentTaskId = document.getElementById('logsModal').getAttribute('data-task-id');
        if (currentTaskId) {
            loadTaskLogsForModal(currentTaskId);
        }
    }
}

// 显示所有任务日志弹窗
function showAllLogsModal() {
    if (!tasks || tasks.length === 0) {
        showToast('没有任务可查看', 'warning');
        return;
    }
    
    // 设置当前任务信息
    document.getElementById('logsTaskName').textContent = '所有任务';
    document.getElementById('logsTaskUrl').textContent = '系统日志';
    document.getElementById('logsTaskStatus').textContent = '系统';
    
    // 设置当前任务ID为特殊值
    document.getElementById('logsModal').setAttribute('data-task-id', 'all');
    document.getElementById('logsTaskStatus').className = 'badge bg-info';
    
    // 清空日志容器
    const logContainer = document.getElementById('logsContainer');
    logContainer.innerHTML = '';
    
    // 显示弹窗
    new bootstrap.Modal(document.getElementById('logsModal')).show();
    
    // 加载所有任务日志
    loadAllTasksLogs();
}

// 加载所有任务日志
async function loadAllTasksLogs() {
    try {
        const allLogs = [];
        for (const task of tasks) {
            try {
                const response = await fetch(API_BASE + `/tasks/${task.id}/logs`);
                if (response.ok) {
                    const logs = await response.json();
                    logs.forEach(log => {
                        allLogs.push(`[${task.name}] ${log}`);
                    });
                }
            } catch (error) {
                console.error(`加载任务 ${task.name} 日志失败:`, error);
            }
        }
        
        // 按时间排序
        allLogs.sort();
        renderLogsInModal(allLogs);
    } catch (error) {
        console.error('加载所有日志失败:', error);
        showToast('加载日志失败: ' + error.message, 'error');
    }
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
        
        // 如果当前任务日志弹窗打开，实时更新显示
        const currentTaskId = document.getElementById('logsModal').getAttribute('data-task-id');
        if (currentTaskId === taskId || currentTaskId === 'all') {
            addLogEntryToModal(logEntry);
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
    if (eventSource) {
        eventSource.close();
    }
});