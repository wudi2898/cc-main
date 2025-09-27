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
	ScheduleInterval int        `json:"schedule_interval"`
	ScheduleDuration int        `json:"schedule_duration"`
	Status           TaskStatus `json:"status"`
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
	StartTime   time.Time
}

// 全局变量
var (
	tasks        = make(map[string]*Task)
	tasksMutex   sync.RWMutex
	tasksFile    = "/cc-tasks.json"
	port         = "8080"
	serverStats  = &ServerStats{StartTime: time.Now()}
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
	
	// 静态文件服务（放在最后，避免拦截API请求）
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/")))
	
	// 启动服务器
	fmt.Println("🚀 API服务器启动中...")
	
	// 启动性能监控
	go updateServerStats()
	
	// 获取服务器IP地址
	serverIP := "localhost"
	if output, err := exec.Command("hostname", "-I").Output(); err == nil && len(output) > 0 {
		ips := strings.Fields(string(output))
		if len(ips) > 0 {
			serverIP = ips[0]
		}
	}
	
	fmt.Printf("📱 前端地址: http://%s:%s\n", serverIP, port)
	fmt.Printf("🔗 API地址: http://%s:%s/api\n", serverIP, port)
	fmt.Printf("📊 日志页面: http://%s:%s/logs.html\n", serverIP, port)
	
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
		log.Printf("❌ 创建任务失败 - JSON解析错误: %v", err)
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 验证必填字段
	if task.Name == "" {
		log.Printf("❌ 创建任务失败 - 任务名称为空")
		http.Error(w, "任务名称不能为空", http.StatusBadRequest)
		return
	}
	if task.TargetURL == "" {
		log.Printf("❌ 创建任务失败 - 目标URL为空")
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
		log.Printf("❌ 保存任务失败: %v", err)
		http.Error(w, "保存任务失败", http.StatusInternalServerError)
		return
	}
	
	log.Printf("✅ 任务创建成功: %s (%s)", task.Name, task.ID)
	
	// 如果状态是running，立即启动
	if task.Status == StatusRunning {
		go startTaskProcess(&task)
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
	
	tasksMutex.Unlock()
	
	// 保存任务列表
	if err := saveTasks(); err != nil {
		log.Printf("❌ 保存任务失败: %v", err)
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
	
	delete(tasks, taskId)
	tasksMutex.Unlock()
	
	// 保存任务列表
	if err := saveTasks(); err != nil {
		log.Printf("❌ 保存任务失败: %v", err)
	}
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func startTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	log.Printf("🚀 尝试启动任务: %s", taskId)
	
	tasksMutex.Lock()
	task, exists := tasks[taskId]
	if !exists {
		tasksMutex.Unlock()
		log.Printf("❌ 任务不存在: %s", taskId)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	if task.Status == StatusRunning {
		tasksMutex.Unlock()
		log.Printf("⚠️  任务已在运行: %s", taskId)
		http.Error(w, "Task is already running", http.StatusBadRequest)
		return
	}
	
	// 验证任务参数
	if task.TargetURL == "" {
		tasksMutex.Unlock()
		log.Printf("❌ 任务目标URL为空: %s", taskId)
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
		log.Printf("❌ 保存任务失败: %v", err)
	}
	
	log.Printf("✅ 任务启动成功: %s -> %s", task.Name, task.TargetURL)
	
	// 启动任务进程
	go startTaskProcess(task)
	
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
	}
	
	task.Status = StatusStopped
	now := time.Now()
	task.CompletedAt = &now
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
	log.Printf("🔧 构建命令参数: %s", task.TargetURL)
	log.Printf("🔧 任务数据: URL=%s, Mode=%s, RandomPath=%v", task.TargetURL, task.Mode, task.RandomPath)
	
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
	
	log.Printf("🔧 执行命令: %s", strings.Join(cmd.Args, " "))
	
	
	// 设置工作目录
	cmd.Dir = "."
	
	// 设置进程组，便于管理
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	
	// 检查cc-go文件是否存在
	if _, err := os.Stat("./cc-go"); os.IsNotExist(err) {
		log.Printf("❌ cc-go文件不存在: %v", err)
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 错误: cc-go文件不存在", time.Now().Format("15:04:05")))
		return
	}
	
	// 设置输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ 创建输出管道失败: %v", err)
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 创建输出管道失败: %v", time.Now().Format("15:04:05"), err))
		return
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("❌ 创建错误管道失败: %v", err)
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 创建错误管道失败: %v", time.Now().Format("15:04:05"), err))
		return
	}
	
	// 启动进程
	task.Process = cmd
	err = cmd.Start()
	if err != nil {
		log.Printf("❌ 启动进程失败: %v", err)
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("[%s] 启动失败: %v", time.Now().Format("15:04:05"), err))
		return
	}
	
	log.Printf("✅ 进程启动成功，PID: %d", cmd.Process.Pid)
	task.Logs = append(task.Logs, fmt.Sprintf("[%s] 进程启动成功，PID: %d", time.Now().Format("15:04:05"), cmd.Process.Pid))
	
	// 启动日志捕获
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line))
			log.Printf("📝 任务日志: %s", line)
			
			// 解析统计信息
			if strings.Contains(line, "STATS_JSON:") {
				statsJSON := strings.TrimPrefix(line, "STATS_JSON:")
				var stats TaskStats
				if err := json.Unmarshal([]byte(statsJSON), &stats); err == nil {
					task.Stats = &stats
				}
			}
		}
	}()
	
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] ERROR: %s", time.Now().Format("15:04:05"), line))
			log.Printf("❌ 任务错误: %s", line)
		}
	}()
	
	// 异步等待进程完成
	go func() {
		err := cmd.Wait()
		if err != nil {
			task.Status = StatusFailed
			task.Logs = append(task.Logs, fmt.Sprintf("进程异常退出: %v", err))
		} else {
			task.Status = StatusCompleted
			task.Logs = append(task.Logs, "任务完成")
		}
		
		now := time.Now()
		task.CompletedAt = &now
	}()
}

// 解析命令行参数
func parseArgs() {
	flag.StringVar(&port, "port", "8080", "服务器端口")
	flag.StringVar(&tasksFile, "tasks-file", "./cc-tasks.json", "任务列表文件路径")
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
		log.Printf("📝 任务文件不存在，创建空列表")
		saveTasks()
		return
	}
	
	// 读取文件
	data, err := ioutil.ReadFile(tasksFile)
	if err != nil {
		log.Printf("❌ 读取任务文件失败: %v", err)
		return
	}
	
	// 解析JSON
	var taskList []*Task
	if err := json.Unmarshal(data, &taskList); err != nil {
		log.Printf("❌ 解析任务文件失败: %v", err)
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
		log.Printf("🔄 已将 %d 个任务状态改为停止", modifiedCount)
	}
	
	log.Printf("✅ 加载了 %d 个任务", len(taskList))
}

// 停止所有运行中的任务
func stopAllRunningTasks() {
	tasksMutex.Lock()
	defer tasksMutex.Unlock()
	
	stoppedCount := 0
	for _, task := range tasks {
		if task.Status == StatusRunning && task.Process != nil {
			log.Printf("🛑 停止运行中的任务: %s (PID: %d)", task.Name, task.Process.Process.Pid)
			
			// 停止进程
			if err := task.Process.Process.Kill(); err != nil {
				log.Printf("❌ 停止任务失败: %v", err)
			} else {
				log.Printf("✅ 任务已停止: %s", task.Name)
				stoppedCount++
			}
			
			// 更新任务状态
			task.Status = StatusStopped
			task.Process = nil
			task.Logs = append(task.Logs, fmt.Sprintf("[%s] 服务重启，任务已停止", time.Now().Format("15:04:05")))
		}
	}
	
	if stoppedCount > 0 {
		log.Printf("🔄 已停止 %d 个运行中的任务", stoppedCount)
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
		log.Printf("序列化任务失败: %v", err)
		return err
	}
	
	// 写入文件
	if err := ioutil.WriteFile(tasksFile, data, 0644); err != nil {
		log.Printf("保存任务文件失败: %v", err)
		return err
	}
	
	return nil
}

// 获取服务器性能统计
func getServerStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(serverStats)
}

// 更新服务器性能统计
func updateServerStats() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
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
		
		// 简单的CPU使用率估算（基于Goroutine数量）
		// 注意：这是一个简化的估算，实际CPU使用率需要更复杂的计算
		serverStats.CPUUsage = float64(serverStats.Goroutines) / 100.0
		if serverStats.CPUUsage > 100 {
			serverStats.CPUUsage = 100
		}
	}
}
