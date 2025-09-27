# CC压力测试工具 - 技术实现文档

## 🔧 技术架构深度解析

### 核心设计理念

本项目采用**微服务架构**设计，将压力测试引擎、API服务器和Web前端完全分离，通过RESTful API和SSE进行通信，实现了高内聚、低耦合的系统架构。

### 系统架构图
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web前端界面   │    │   API服务器     │    │   压力测试引擎   │
│                 │    │                 │    │                 │
│  - 任务管理     │◄──►│  - RESTful API  │◄──►│  - 高并发引擎   │
│  - 实时监控     │    │  - 任务调度     │    │  - 代理池管理   │
│  - 图表展示     │    │  - 数据持久化   │    │  - CF绕过      │
│  - 响应式设计   │    │  - SSE推送      │    │  - 统计收集    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 技术栈选择理由

#### 后端技术选型
- **Go语言**: 高并发性能优异，内存管理高效
- **Gorilla Mux**: 轻量级路由，性能优于标准库
- **Gorilla WebSocket**: 成熟的WebSocket实现
- **Fake User-Agent**: 专业的UA伪造库

## 🏗️ 核心模块技术实现

### 1. 压力测试引擎 (main.go)

#### 并发控制机制
```go
// 协程池实现
type WorkerPool struct {
    workers    int
    jobs       chan Job
    results    chan Result
    wg         sync.WaitGroup
}

// 高并发请求处理
func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        go p.worker()
    }
}
```

#### 性能优化策略
- **连接复用**: 使用HTTP Client连接池
- **内存池**: 预分配缓冲区减少GC压力
- **协程控制**: 限制最大协程数量防止资源耗尽
- **批量处理**: 批量发送请求提高吞吐量

#### CF绕过技术实现
```go
// 随机HTTP头生成
func generateRandomHeaders() map[string]string {
    headers := make(map[string]string)
    
    // 随机User-Agent
    headers["User-Agent"] = fakeuseragent.Random()
    
    // 随机Referer
    headers["Referer"] = generateRandomReferer()
    
    // 现代浏览器头
    headers["Sec-Ch-Ua"] = generateSecChUa()
    headers["Sec-Fetch-Dest"] = "document"
    headers["Sec-Fetch-Mode"] = "navigate"
    
    return headers
}
```

### 2. API服务器架构 (api_server.go)

#### 任务调度器设计
```go
// 定时任务调度器
type Scheduler struct {
    tasks      map[string]*Task
    tickers    map[string]*time.Ticker
    mutex      sync.RWMutex
}

// 定时任务执行
func (s *Scheduler) StartScheduledTask(task *Task) {
    ticker := time.NewTicker(time.Duration(task.ScheduleInterval) * time.Minute)
    s.tickers[task.ID] = ticker
    
    go func() {
        for range ticker.C {
            s.executeTask(task)
        }
    }()
}
```

#### 实时数据推送机制
```go
// SSE连接管理
type SSEManager struct {
    connections map[string]http.ResponseWriter
    mutex       sync.RWMutex
}

// 广播消息
func (s *SSEManager) Broadcast(data interface{}) {
    jsonData, _ := json.Marshal(data)
    message := fmt.Sprintf("data: %s\n\n", jsonData)
    
    s.mutex.RLock()
    for _, conn := range s.connections {
        fmt.Fprintf(conn, message)
        if flusher, ok := conn.(http.Flusher); ok {
            flusher.Flush()
        }
    }
    s.mutex.RUnlock()
}
```

#### 数据持久化策略
```go
// 任务持久化
func saveTasks() error {
    tasksMutex.RLock()
    var taskList []*Task
    for _, task := range tasks {
        taskList = append(taskList, task)
    }
    tasksMutex.RUnlock()
    
    data, err := json.MarshalIndent(taskList, "", "  ")
    if err != nil {
        return err
    }
    
    return ioutil.WriteFile(tasksFile, data, 0644)
}
```

### 3. 前端架构设计 (frontend/)

#### 实时图表系统
```javascript
// 图表初始化
function initCharts() {
    // CPU使用率图表
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
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: { beginAtZero: true, max: 100 },
                x: { display: false }
            }
        }
    });
}
```

#### SSE实时通信
```javascript
// SSE连接管理
function initSSE() {
    eventSource = new EventSource(API_BASE + '/events');
    
    eventSource.onmessage = function(event) {
        const data = JSON.parse(event.data);
        if (data.type === 'task_update') {
            updateTaskInList(data.task);
        } else if (data.type === 'task_log') {
            addLogEntry(data.task_id, data.log);
        }
    };
}
```

#### 任务状态管理
```javascript
// 任务状态更新
function updateTaskCard(cardElement, task) {
    const statusClass = getStatusClass(task.status);
    const statusText = getStatusText(task.status);
    
    // 更新状态显示
    const statusElement = cardElement.querySelector('.task-status');
    if (statusElement) {
        statusElement.className = `task-status ${statusClass}`;
        statusElement.innerHTML = `<i class="bi bi-${getStatusIcon(task.status)}"></i> ${statusText}`;
    }
    
    // 更新统计信息
    if (task.stats) {
        updateTaskStats(cardElement, task.stats);
    }
}
```

## 🔧 核心技术实现细节

### 1. 高并发处理机制

#### 协程池设计
```go
// 工作协程池
type WorkerPool struct {
    workerCount int
    jobQueue    chan *Job
    resultQueue chan *Result
    wg          sync.WaitGroup
    ctx         context.Context
    cancel      context.CancelFunc
}

// 启动工作池
func (p *WorkerPool) Start() {
    for i := 0; i < p.workerCount; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

// 工作协程
func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()
    
    for {
        select {
        case job := <-p.jobQueue:
            result := p.processJob(job)
            p.resultQueue <- result
        case <-p.ctx.Done():
            return
        }
    }
}
```

#### 内存优化策略
```go
// 对象池模式
var requestPool = sync.Pool{
    New: func() interface{} {
        return &http.Request{}
    },
}

// 使用对象池
func makeRequest(url string) (*http.Response, error) {
    req := requestPool.Get().(*http.Request)
    defer requestPool.Put(req)
    
    // 重置请求对象
    req.URL, _ = url.Parse(url)
    req.Method = "GET"
    req.Header = make(http.Header)
    
    return client.Do(req)
}
```

### 2. 网络性能优化

#### 连接池配置
```go
// HTTP客户端优化
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        1000,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        DisableKeepAlives:   false,
    },
    Timeout: 30 * time.Second,
}
```

#### 代理轮换机制
```go
// 代理池管理
type ProxyPool struct {
    proxies []string
    current int
    mutex   sync.Mutex
}

// 获取下一个代理
func (p *ProxyPool) GetNext() string {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    if len(p.proxies) == 0 {
        return ""
    }
    
    proxy := p.proxies[p.current]
    p.current = (p.current + 1) % len(p.proxies)
    return proxy
}
```

### 3. 实时监控系统

#### 系统性能采集
```go
// 系统监控
func updateServerStats() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        // CPU使用率
        serverStats.CPUUsage = calculateCPUUsage()
        
        // 内存使用率
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        serverStats.MemoryUsage = float64(m.Alloc) / float64(m.Sys) * 100
        
        // 网络速度
        updateNetworkStats()
        
        // 推送更新
        sendSSEMessage(map[string]interface{}{
            "type": "stats_update",
            "data": serverStats,
        })
    }
}
```

#### 网络速度监控
```go
// 网络统计
func updateNetworkStats() {
    data, err := ioutil.ReadFile("/proc/net/dev")
    if err != nil {
        return
    }
    
    lines := strings.Split(string(data), "\n")
    var totalRx, totalTx uint64
    
    for _, line := range lines {
        if strings.Contains(line, ":") && !strings.Contains(line, "lo:") {
            parts := strings.Fields(line)
            if len(parts) >= 10 {
                if rx, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
                    totalRx += rx
                }
                if tx, err := strconv.ParseUint(parts[9], 10, 64); err == nil {
                    totalTx += tx
                }
            }
        }
    }
    
    // 计算速度
    now := time.Now()
    if !lastNetTime.IsZero() {
        timeDiff := now.Sub(lastNetTime).Seconds()
        if timeDiff > 0 {
            rxDiff := float64(totalRx - lastRxBytes)
            txDiff := float64(totalTx - lastTxBytes)
            
            serverStats.NetworkRx = (rxDiff / (1024 * 1024)) / timeDiff
            serverStats.NetworkTx = (txDiff / (1024 * 1024)) / timeDiff
        }
    }
    
    lastRxBytes = totalRx
    lastTxBytes = totalTx
    lastNetTime = now
}
```

## 🚀 快速开始

### 环境要求

- **操作系统**: Linux/macOS/Windows
- **Go版本**: 1.21+
- **内存**: 建议2GB+
- **网络**: 稳定的网络连接

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd cc-main
```

2. **安装依赖**
```bash
./install.sh
```

3. **启动服务**
```bash
./start.sh
```

4. **访问控制面板**
```
http://localhost:8080
```

### 管理命令

```bash
# 启动服务
./start.sh

# 停止服务
./stop.sh

# 查看状态
./status.sh

# 查看日志
tail -f api_server.log
```

## 🔧 核心功能详解

### 1. 压力测试引擎 (main.go)

#### 主要特性
- **高并发支持**: 支持数万级并发连接
- **多种攻击模式**: GET、POST、HEAD等HTTP方法
- **智能重试**: 自动重试失败请求
- **统计收集**: 实时统计请求成功率、RPS等

#### 核心配置
```go
type Config struct {
    TargetURL         string  // 目标URL
    Mode              string  // 攻击模式
    Threads           int     // 线程数
    RPS               int     // 每秒请求数
    Duration          int     // 持续时间(秒)
    Timeout           int     // 超时时间(秒)
    ProxyFile         string  // 代理文件路径
    CFBypass          bool    // CF绕过
    RandomPath        bool    // 随机路径
    RandomParams      bool    // 随机参数
    Schedule          bool    // 定时执行
    ScheduleInterval  int     // 定时间隔(分钟)
    ScheduleDuration  int     // 执行时长(分钟)
    FireAndForget     bool    // 火后不理模式
}
```

#### 性能优化
- **连接池**: 复用HTTP连接
- **协程池**: 控制并发数量
- **内存优化**: 减少内存分配
- **CPU优化**: 多核并行处理

### 2. API服务器 (api_server.go)

#### 主要功能
- **任务管理**: CRUD操作
- **实时监控**: 系统性能统计
- **定时调度**: 定时任务管理
- **数据持久化**: 任务列表保存
- **SSE推送**: 实时数据推送

#### API接口

##### 任务管理
```
GET    /api/tasks              # 获取任务列表
POST   /api/tasks              # 创建任务
GET    /api/tasks/{id}         # 获取任务详情
PUT    /api/tasks/{id}         # 更新任务
DELETE /api/tasks/{id}         # 删除任务
POST   /api/tasks/{id}/start   # 启动任务
POST   /api/tasks/{id}/stop    # 停止任务
GET    /api/tasks/{id}/logs    # 获取任务日志
GET    /api/tasks/{id}/stats   # 获取任务统计
```

##### 系统监控
```
GET    /api/server-stats       # 服务器性能统计
GET    /api/traffic-stats      # 流量统计
GET    /api/events             # SSE事件流
```

#### 任务状态管理
```go
type TaskStatus string

const (
    StatusPending   TaskStatus = "pending"   // 待启动
    StatusRunning   TaskStatus = "running"   // 运行中
    StatusCompleted TaskStatus = "completed" // 已完成
    StatusFailed    TaskStatus = "failed"    // 失败
    StatusStopped   TaskStatus = "stopped"   // 已停止
)
```

### 3. Web控制面板 (frontend/)

#### 界面特性
- **响应式设计**: 适配各种屏幕尺寸
- **现代化UI**: 使用Bootstrap 5框架
- **实时更新**: 2秒自动刷新
- **图表展示**: 6个实时监控图表

#### 主要页面
- **任务列表**: 显示所有任务状态
- **任务创建**: 创建新的压力测试任务
- **任务编辑**: 修改任务配置
- **实时监控**: 系统性能图表
- **日志查看**: 任务执行日志

#### 实时图表
1. **CPU使用率**: 服务器CPU使用情况
2. **内存使用率**: 服务器内存使用情况
3. **实时流量**: 总请求数统计
4. **Goroutines**: Go协程数量
5. **网络接收速度**: 网卡接收速度
6. **网络发送速度**: 网卡发送速度

## ⚙️ 配置说明

### 服务器配置

#### 启动参数
```bash
./api_server --port=8080 --tasks-file=/cc-tasks.json
```

#### 环境变量
```bash
export CC_PORT=8080
export CC_TASKS_FILE=/cc-tasks.json
```

### 任务配置

#### 基础配置
- **任务名称**: 自定义任务名称
- **目标URL**: 压力测试目标地址
- **攻击模式**: GET/POST/HEAD
- **线程数**: 并发连接数 (1-100000)
- **RPS**: 每秒请求数 (1-1000000)
- **持续时间**: 测试时长 (1-86400秒)
- **超时时间**: 请求超时 (1-300秒)

#### 高级配置
- **CF绕过**: Cloudflare防护绕过
- **随机路径**: 随机URL路径
- **随机参数**: 随机查询参数
- **定时执行**: 启用定时任务
- **执行间隔**: 定时间隔 (1-1440分钟)
- **执行时长**: 每次执行时长 (1-1440分钟)

### 代理配置

#### SOCKS5代理
- 支持SOCKS5代理池
- 代理文件: `socks5.txt`
- 格式: `ip:port:username:password`
- 自动轮换代理

## 📊 性能监控

### 系统监控指标

#### 服务器性能
- **CPU使用率**: 实时CPU占用
- **内存使用率**: 内存占用百分比
- **Goroutines**: Go协程数量
- **运行时间**: 服务器运行时长

#### 网络监控
- **接收速度**: 网络接收速度 (MB/s)
- **发送速度**: 网络发送速度 (MB/s)
- **总请求数**: 累计请求数量
- **成功率**: 请求成功百分比

#### 任务统计
- **当前RPS**: 实时每秒请求数
- **平均RPS**: 平均每秒请求数
- **成功请求**: 成功请求数量
- **失败请求**: 失败请求数量

### 图表更新频率
- **自动刷新**: 每2秒更新一次
- **实时推送**: SSE事件流
- **历史数据**: 保留最近30个数据点

## 🔒 安全特性

### 防护机制
- **请求限制**: 防止过度请求
- **超时控制**: 避免长时间占用
- **资源监控**: 实时监控系统资源
- **错误处理**: 完善的错误处理机制

### 代理支持
- **IP轮换**: 自动切换代理IP
- **用户代理**: 随机User-Agent
- **请求头**: 模拟真实浏览器
- **Cookie支持**: 会话保持

## 🛠️ 开发指南

### 代码结构

#### 后端架构
```
api_server.go
├── 任务管理模块
│   ├── 任务CRUD操作
│   ├── 任务状态管理
│   └── 任务持久化
├── 调度器模块
│   ├── 定时任务调度
│   ├── 任务生命周期管理
│   └── 调度器清理
├── 监控模块
│   ├── 系统性能监控
│   ├── 网络速度监控
│   └── 实时数据推送
└── API接口模块
    ├── RESTful API
    ├── SSE事件流
    └── 错误处理
```

#### 前端架构
```
app.js
├── 图表管理模块
│   ├── 图表初始化
│   ├── 数据更新
│   └── 图表配置
├── 任务管理模块
│   ├── 任务列表渲染
│   ├── 任务操作
│   └── 表单处理
├── 实时通信模块
│   ├── SSE连接管理
│   ├── 事件处理
│   └── 数据同步
└── 工具函数模块
    ├── 数据格式化
    ├── 错误处理
    └── 用户交互
```

### 扩展开发

#### 添加新的监控指标
1. 在`ServerStats`结构体中添加新字段
2. 在`updateServerStats()`函数中实现数据收集
3. 在前端添加对应的图表
4. 更新`updateCharts()`函数

#### 添加新的攻击模式
1. 在`main.go`中添加新的HTTP方法支持
2. 在`Config`结构体中添加配置选项
3. 在前端表单中添加对应的选择项
4. 更新API接口处理逻辑

#### 添加新的代理类型
1. 在`main.go`中实现新的代理协议
2. 添加代理配置解析逻辑
3. 更新代理轮换机制
4. 添加代理健康检查

## 🐛 故障排除

### 常见问题

#### 1. 服务启动失败
**问题**: API服务器无法启动
**解决方案**:
```bash
# 检查端口占用
netstat -tuln | grep 8080

# 检查Go环境
go version

# 查看错误日志
cat api_server.log
```

#### 2. 任务执行失败
**问题**: 压力测试任务无法启动
**解决方案**:
- 检查目标URL是否可访问
- 验证网络连接
- 检查代理配置
- 查看任务日志

#### 3. 图表不显示
**问题**: 实时监控图表无数据
**解决方案**:
- 检查浏览器控制台错误
- 验证API接口响应
- 检查SSE连接状态
- 刷新页面重试

#### 4. 定时任务不执行
**问题**: 定时任务未按计划执行
**解决方案**:
- 检查定时间隔配置
- 验证任务状态
- 查看调度器日志
- 重启服务

### 日志分析

#### 服务器日志
```bash
# 查看实时日志
tail -f api_server.log

# 搜索错误信息
grep -i error api_server.log

# 查看任务执行日志
grep "任务" api_server.log
```

#### 系统监控
```bash
# 查看进程状态
./status.sh

# 监控系统资源
top -p $(pgrep api_server)

# 查看网络连接
netstat -an | grep 8080
```

## 📈 性能优化

### 系统优化

#### 服务器优化
- **增加文件描述符限制**: `ulimit -n 65536`
- **优化TCP参数**: 调整内核网络参数
- **内存优化**: 使用内存池减少GC压力
- **CPU优化**: 绑定CPU核心提高性能

#### 网络优化
- **连接复用**: 启用HTTP Keep-Alive
- **代理优化**: 使用高质量代理
- **DNS优化**: 使用快速DNS服务器
- **带宽优化**: 合理设置并发数

### 应用优化

#### 后端优化
- **协程池**: 限制协程数量
- **内存池**: 减少内存分配
- **缓存优化**: 缓存频繁访问的数据
- **数据库优化**: 优化查询性能

#### 前端优化
- **资源压缩**: 压缩CSS/JS文件
- **缓存策略**: 设置合适的缓存头
- **懒加载**: 延迟加载非关键资源
- **CDN加速**: 使用CDN分发静态资源

## 🔄 版本更新

### 版本历史

#### v2.0.0 (当前版本)
- ✅ 添加Web控制面板
- ✅ 实现实时监控图表
- ✅ 支持定时任务调度
- ✅ 添加网络速度监控
- ✅ 优化任务持久化
- ✅ 改进错误处理机制

#### v1.x.x (历史版本)
- 基础压力测试功能
- 命令行界面
- 基础代理支持

### 升级指南

#### 从v1.x升级到v2.0
1. 备份现有配置
2. 停止旧版本服务
3. 下载新版本代码
4. 运行安装脚本
5. 启动新版本服务
6. 验证功能正常

#### 配置迁移
- 任务配置自动兼容
- 代理文件格式不变
- 日志格式保持兼容

## 📞 技术支持

### 联系方式
- **项目仓库**: [GitHub Repository]
- **问题反馈**: [Issues Page]
- **技术讨论**: [Discussions Page]

### 贡献指南
1. Fork项目仓库
2. 创建功能分支
3. 提交代码更改
4. 创建Pull Request
5. 等待代码审查

### 开发规范
- **代码风格**: 遵循Go官方规范
- **提交信息**: 使用清晰的提交信息
- **测试覆盖**: 确保新功能有测试
- **文档更新**: 及时更新相关文档

## 📄 许可证

本项目采用 [MIT License] 许可证，详情请查看 LICENSE 文件。

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者和用户。

---

**注意**: 本工具仅用于合法的压力测试和性能评估，请遵守相关法律法规，不得用于非法用途。
