package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
)

// fetchJSONAPI 抓取 JSON API 源
func fetchJSONAPI(ctx context.Context, source config.Source, client *http.Client) (*FetchResult, error) {
	logger := slog.With("source", source.Name)
	logger.Info("开始抓取 JSON API", "url", source.URL)

	jc := source.JSONConfig
	if jc == nil {
		return nil, fmt.Errorf("json_config 未配置")
	}

	// 构建请求
	method := strings.ToUpper(source.Method)
	if method == "" {
		method = http.MethodGet
	}

	var req *http.Request
	var err error

	if method == http.MethodPost && len(source.Body) > 0 {
		req, err = http.NewRequestWithContext(ctx, method, source.URL, bytes.NewReader(source.Body))
		if err != nil {
			return nil, fmt.Errorf("创建 POST 请求失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建 GET 请求失败: %w", err)
		}
	}

	applyHeaders(req, source.Headers)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 解析 JSON
	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 通过路径提取条目列表
	rawItems := getNestedValue(data, jc.ItemsPath)
	items, ok := rawItems.([]any)
	if !ok {
		return nil, fmt.Errorf("items_path '%s' 未返回列表", jc.ItemsPath)
	}

	// 映射字段
	titleField := jc.TitleField
	if titleField == "" {
		titleField = "title"
	}

	entries := make([]Entry, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		title := getString(item, titleField)
		if title == "" {
			continue
		}

		// 构建链接
		link := buildLink(item, jc.LinkField, jc.LinkTemplate)

		// 摘要
		summary := ""
		if jc.SummaryField != "" {
			summary = getString(item, jc.SummaryField)
		}

		// 时间解析
		var pubTime time.Time
		if jc.TimeField != "" {
			pubTime = parseTimeString(getString(item, jc.TimeField))
		}

		entries = append(entries, Entry{
			Title:      title,
			Link:       link,
			Summary:    summary,
			Published:  pubTime,
			SourceType: "json_api",
			RawData:    item,
		})
	}

	logger.Info("JSON API 抓取完成", "count", len(entries))

	return &FetchResult{
		Source:    source,
		Entries:   entries,
		FetchedAt: time.Now().UTC(),
	}, nil
}

// getNestedValue 通过点号分隔的路径获取嵌套 JSON 值
// 例如: "data.realtime" 从 {"data": {"realtime": [...]}} 获取列表
func getNestedValue(data any, path string) any {
	if path == "" {
		return data
	}

	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		switch v := current.(type) {
		case map[string]any:
			current = v[key]
		default:
			return nil
		}
		if current == nil {
			return nil
		}
	}

	return current
}

// buildLink 构建条目链接，支持直接字段名或模板格式
func buildLink(item map[string]any, linkField, linkTemplate string) string {
	if linkTemplate != "" {
		// 模板格式: "https://example.com/{id}"
		link := linkTemplate
		for key, value := range item {
			placeholder := "{" + key + "}"
			if strings.Contains(link, placeholder) {
				link = strings.ReplaceAll(link, placeholder, fmt.Sprintf("%v", value))
			}
		}
		return link
	}

	if linkField != "" {
		return getString(item, linkField)
	}

	return ""
}

// getString 安全地从 map 中获取字符串值
func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// parseTimeString 尝试多种格式解析时间字符串
func parseTimeString(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	// 尝试 Unix 时间戳（秒）
	if len(s) == 10 {
		var ts int64
		if _, err := fmt.Sscanf(s, "%d", &ts); err == nil && ts > 1000000000 {
			return time.Unix(ts, 0).UTC()
		}
	}

	// 常见时间格式
	formats := []string{
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}

	return time.Time{}
}
