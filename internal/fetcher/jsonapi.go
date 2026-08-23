package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/timeutil"
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

	resp, err := doRequestWithRetry(ctx, client, source, method)
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

	// 时间解析：无时区字符串按配置时区（默认北京时间）解析
	timeLoc := timeutil.ResolveLocation(jc.TimeZone)

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
			pubTime = parseTimeString(getString(item, jc.TimeField), timeLoc)
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

// buildLink 构建条目链接，支持直接字段名或模板格式（支持嵌套占位符如 {author.id}）
func buildLink(item map[string]any, linkField, linkTemplate string) string {
	if linkTemplate != "" {
		link := linkTemplate
		queryIdx := strings.Index(linkTemplate, "?")
		
		// 提取所有形如 {xxx} 的占位符，包括点号嵌套路径
		var placeholders []string
		start := -1
		for i, r := range linkTemplate {
			if r == '{' {
				start = i
			} else if r == '}' && start != -1 {
				placeholders = append(placeholders, linkTemplate[start:i+1])
				start = -1
			}
		}

		for _, placeholder := range placeholders {
			path := placeholder[1 : len(placeholder)-1] // 移除两端花括号
			value := getNestedValue(item, path)
			if value != nil {
				valStr := fmt.Sprintf("%v", value)
				placeholderIdx := strings.Index(linkTemplate, placeholder)
				
				var escapedVal string
				// 智能自动编码：当占位符在 Query 部分（问号之后），采用 QueryEscape；否则采用 PathEscape
				if queryIdx != -1 && placeholderIdx > queryIdx {
					escapedVal = url.QueryEscape(valStr)
				} else {
					escapedVal = url.PathEscape(valStr)
				}
				
				link = strings.ReplaceAll(link, placeholder, escapedVal)
			}
		}
		return link
	}

	if linkField != "" {
		return getString(item, linkField)
	}

	return ""
}

// getString 安全地从 map 中获取字符串值，支持以点号 "." 分隔的嵌套路径（例如 "detail.title"）
func getString(m map[string]any, key string) string {
	var v any
	if strings.Contains(key, ".") {
		v = getNestedValue(m, key)
	} else {
		var ok bool
		v, ok = m[key]
		if !ok {
			return ""
		}
	}
	
	if v == nil {
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

// parseTimeString 尝试多种格式解析时间字符串。
// 带时区的格式按其自身时区解析；无时区的格式按 loc 指定时区解析
// （中文源接口返回的本地时间多为北京时间）。
func parseTimeString(s string, loc *time.Location) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}

	// 尝试纯数字 Unix 时间戳：13 位毫秒 / 10 位秒
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		switch {
		case len(s) >= 13 && ts > 1000000000000:
			return time.UnixMilli(ts).UTC()
		case len(s) == 10 && ts > 1000000000:
			return time.Unix(ts, 0).UTC()
		}
	}

	// 常见带时区的时间格式（按字符串自身时区）
	formats := []string{
		time.RFC3339Nano, // 2006-01-02T15:04:05.999999999Z07:00，兼容带毫秒的 ISO8601
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 2006 15:04:05 -0700",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}

	// 无时区格式：按指定时区（默认北京时间）解析
	localFormats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02",
	}
	for _, f := range localFormats {
		if t, err := time.ParseInLocation(f, s, loc); err == nil {
			return t.UTC()
		}
	}

	return time.Time{}
}
