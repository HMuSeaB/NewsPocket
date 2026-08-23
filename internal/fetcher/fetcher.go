// Package fetcher 负责并发抓取所有新闻源。
// 统一 RSS 和 JSON API 两种源类型的抓取结果为 Entry 格式。
package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
)

// Entry 统一的新闻条目格式，所有源类型（RSS/JSON API）转换为此结构
type Entry struct {
	Title      string         `json:"title"`
	Link       string         `json:"link"`
	Summary    string         `json:"summary"`
	Published  time.Time      `json:"published"`
	SourceType string         `json:"source_type"`        // "rss" | "json_api"
	RawData    map[string]any `json:"raw_data,omitempty"` // 调试用原始数据
}

// FetchResult 单个源的抓取结果
type FetchResult struct {
	Source    config.Source
	Entries   []Entry
	FetchedAt time.Time
	Err       error
}

// Fetcher 并发抓取编排器
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// New 创建抓取器实例
func New(timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout + 5*time.Second, // 比 context 超时稍长，优先让 context 控制
		},
		timeout: timeout,
	}
}

// defaultHeaders 返回默认请求头
var defaultHeaders = map[string]string{
	"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Accept":          "application/json, application/rss+xml, application/xml, text/xml, */*",
	"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
}

var retryDelay = func(attempt int) time.Duration {
	return time.Duration(attempt+1) * 2 * time.Second
}

// applyHeaders 设置默认 headers 并覆盖为自定义 headers
func applyHeaders(req *http.Request, custom map[string]string) {
	for k, v := range defaultHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range custom {
		req.Header.Set(k, v)
	}
}

// FetchAll 并发抓取所有启用的源。
// 返回与传入源一一对应的结果列表：失败的源以 Err 字段标记失败原因，
// 成功的源 Err 为 nil（即使条目数为 0）。
func (f *Fetcher) FetchAll(sources []config.Source) []FetchResult {
	if len(sources) == 0 {
		slog.Warn("没有配置任何源")
		return nil
	}

	slog.Info("开始并发抓取", "count", len(sources))

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make([]FetchResult, 0, len(sources))
	)

	// 使用 Channel 信号量限制全局并发数，防止触发反爬或协程风暴
	sem := make(chan struct{}, 15)

	for _, src := range sources {
		wg.Add(1)
		sem <- struct{}{} // 阻塞获取信号量

		go func(s config.Source) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
			defer cancel()

			var result *FetchResult
			var err error

			switch s.Type {
			case "json_api":
				result, err = fetchJSONAPI(ctx, s, f.client)
			default:
				// rss, rsshub 等都走 RSS 解析
				result, err = fetchRSS(ctx, s, f.client)
			}

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				slog.Error("抓取失败", "source", s.Name, "error", err)
				results = append(results, FetchResult{Source: s, Err: err})
				return
			}
			results = append(results, *result)
		}(src)
	}

	wg.Wait()

	failed := 0
	for _, r := range results {
		if r.Err != nil {
			failed++
		}
	}

	slog.Info("抓取完成",
		"success", len(results)-failed,
		"failed", failed,
	)

	return results
}

// FetchSingle 抓取单个源（GUI 测试用）
func (f *Fetcher) FetchSingle(source config.Source) (*FetchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	switch source.Type {
	case "json_api":
		return fetchJSONAPI(ctx, source, f.client)
	default:
		return fetchRSS(ctx, source, f.client)
	}
}

// doRequestWithRetry 包装 HTTP 请求，提供指数退避的自动重试机制
func doRequestWithRetry(ctx context.Context, client *http.Client, source config.Source, method string) (*http.Response, error) {
	var resp *http.Response
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		var bodyReader io.Reader
		if len(source.Body) > 0 {
			bodyReader = bytes.NewReader(source.Body)
		}

		req, reqErr := http.NewRequestWithContext(ctx, method, source.URL, bodyReader)
		if reqErr != nil {
			return nil, fmt.Errorf("创建请求失败: %w", reqErr)
		}

		if method == http.MethodPost && len(source.Body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		applyHeaders(req, source.Headers)

		resp, err = client.Do(req)

		// 成功、明确的 4xx 错误则不再重试；429 限流与 5xx 例外，需退避重试
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		if i < maxRetries-1 {
			if err != nil {
				slog.Warn("请求异常准备重试", "source", source.Name, "retry", i+1, "error", err)
			} else {
				slog.Warn("请求返回异常状态码准备重试", "source", source.Name, "retry", i+1, "status", resp.Status)
			}

			// 429 时优先遵循服务端的 Retry-After 指示（封顶 30s）
			delay := retryDelay(i)
			if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
				if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
					delay = ra
				}
			}

			if resp != nil {
				resp.Body.Close()
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("经过 %d 次重试后失败: %w", maxRetries, err)
	}
	return nil, fmt.Errorf("经过 %d 次重试后仍然失败 (可能由于 5xx 或 429 状态码)", maxRetries)
}

// parseRetryAfter 解析 Retry-After 响应头的秒数表示，非法或缺失返回 0
func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	sec, err := strconv.Atoi(value)
	if err != nil || sec <= 0 {
		return 0
	}
	d := time.Duration(sec) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}
