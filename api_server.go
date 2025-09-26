package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	TargetURL    string     `json:"target_url"`
	Mode         string     `json:"mode"`
	Threads      int        `json:"threads"`
	RPS          int        `json:"rps"`
	Duration     int        `json:"duration"`
	Timeout      int        `json:"timeout"`
	CFBypass     bool       `json:"cf_bypass"`
	RandomPath   bool       `json:"random_path"`
	RandomParams bool       `json:"random_params"`
	Status       TaskStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Process      *exec.Cmd  `json:"-"`
	Logs         []string   `json:"logs"`
	Stats        *TaskStats `json:"stats"`
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

// 全局变量
var (
	tasks      = make(map[string]*Task)
	tasksMutex sync.RWMutex
	upgrader   = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	tasksFile = "/cc-tasks.json"
	port      = "8080"
)

func main() {
	// 解析命令行参数
	parseArgs()
	
	// 加载任务列表
	loadTasks()
	
	// 创建路由器
	r := mux.NewRouter()
	
	// 静态文件服务
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/")))
	
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
	
	// WebSocket连接
	api.HandleFunc("/ws", handleWebSocket)
	
	// 启动服务器
	fmt.Println("🚀 API服务器启动中...")
	fmt.Printf("📱 前端地址: http://localhost:%s\n", port)
	fmt.Printf("🔗 API地址: http://localhost:%s/api\n", port)
	fmt.Printf("📊 日志页面: http://localhost:%s/logs.html\n", port)
	
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
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
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
	saveTasks()
	
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
	
	tasksMutex.Unlock()
	
	// 保存任务列表
	saveTasks()
	
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
	saveTasks()
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func startTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["id"]
	
	tasksMutex.Lock()
	task, exists := tasks[taskId]
	if !exists {
		tasksMutex.Unlock()
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	if task.Status == StatusRunning {
		tasksMutex.Unlock()
		http.Error(w, "Task is already running", http.StatusBadRequest)
		return
	}
	
	task.Status = StatusRunning
	now := time.Now()
	task.StartedAt = &now
	tasksMutex.Unlock()
	
	// 保存任务列表
	saveTasks()
	
	// 启动任务进程
	go startTaskProcess(task)
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
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

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()
	
	// 发送所有任务状态
	tasksMutex.RLock()
	for _, task := range tasks {
		conn.WriteJSON(map[string]interface{}{
			"type": "task_update",
			"task": task,
		})
	}
	tasksMutex.RUnlock()
	
	// 保持连接活跃
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// 启动任务进程
func startTaskProcess(task *Task) {
	// 构建命令
	cmd := exec.Command("./cc-go",
		"-url", task.TargetURL,
		"-mode", task.Mode,
		"-threads", strconv.Itoa(task.Threads),
		"-rps", strconv.Itoa(task.RPS),
		"-duration", strconv.Itoa(task.Duration),
		"-timeout", strconv.Itoa(task.Timeout),
		"-cf-bypass", strconv.FormatBool(task.CFBypass),
		"-random-path", strconv.FormatBool(task.RandomPath),
		"-random-params", strconv.FormatBool(task.RandomParams),
	)
	
	// 设置工作目录
	cmd.Dir = "."
	
	// 设置进程组，便于管理
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	
	// 启动进程
	task.Process = cmd
	err := cmd.Start()
	if err != nil {
		task.Status = StatusFailed
		task.Logs = append(task.Logs, fmt.Sprintf("启动失败: %v", err))
		return
	}
	
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
		log.Printf("读取任务文件失败: %v", err)
		return
	}
	
	// 解析JSON
	var taskList []*Task
	if err := json.Unmarshal(data, &taskList); err != nil {
		log.Printf("解析任务文件失败: %v", err)
		return
	}
	
	// 加载到内存
	tasksMutex.Lock()
	for _, task := range taskList {
		tasks[task.ID] = task
	}
	tasksMutex.Unlock()
	
	log.Printf("✅ 加载了 %d 个任务", len(taskList))
}

// 保存任务列表
func saveTasks() {
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
		return
	}
	
	// 写入文件
	if err := ioutil.WriteFile(tasksFile, data, 0644); err != nil {
		log.Printf("保存任务文件失败: %v", err)
	}
}
