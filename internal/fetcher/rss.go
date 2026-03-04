package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/HMuSeaB/NewsPocket/internal/config"
)

// fetchRSS 抓取 RSS/Atom 源并转换为统一的 FetchResult
func fetchRSS(ctx context.Context, source config.Source, client *http.Client) (*FetchResult, error) {
	logger := slog.With("source", source.Name)
	logger.Info("开始抓取 RSS", "url", source.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置默认和自定义 headers
	applyHeaders(req, source.Headers)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 使用 gofeed 解析
	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("RSS 解析失败: %w", err)
	}

	// 转换为统一格式
	entries := make([]Entry, 0, len(feed.Items))
	for _, item := range feed.Items {
		entry := Entry{
			Title:      item.Title,
			Link:       item.Link,
			SourceType: "rss",
		}

		// 摘要：优先 description，其次 content
		if item.Description != "" {
			entry.Summary = item.Description
		} else if item.Content != "" {
			entry.Summary = item.Content
		}

		// 发布时间
		if item.PublishedParsed != nil {
			entry.Published = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			entry.Published = *item.UpdatedParsed
		}

		entries = append(entries, entry)
	}

	logger.Info("RSS 抓取完成", "count", len(entries))

	return &FetchResult{
		Source:    source,
		Entries:   entries,
		FetchedAt: time.Now().UTC(),
	}, nil
}
