// Package digest 编排「并发抓取 → 解析清洗 → 分类分组 → AI 速览 → 模板渲染」
// 的完整晨报生成流程，供 CLI (cmd/newspocket) 与 GUI (cmd/newspocket-gui) 共用，
// 避免两端各维护一份重复的主流程。
package digest

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/ai"
	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/fetcher"
	"github.com/HMuSeaB/NewsPocket/internal/parser"
	"github.com/HMuSeaB/NewsPocket/internal/renderer"
	"github.com/HMuSeaB/NewsPocket/internal/timeutil"
)

// 无内容类哨兵错误：调用方（CLI/GUI）按业务提示处理，而非视为故障。
var (
	// ErrNoSources 表示配置中没有任何启用的新闻源
	ErrNoSources = errors.New("没有启用的新闻源")
	// ErrNoResults 表示所有源均抓取失败或返回空内容
	ErrNoResults = errors.New("未抓取到任何内容")
	// ErrNoItems 表示抓取到了原始数据，但经时间过滤与清洗后无有效条目
	ErrNoItems = errors.New("解析后无有效内容")
)

// Options 单次晨报生成的可调参数
type Options struct {
	FetchTimeout time.Duration // 单个源的抓取超时时间
	TitleSuffix  string        // 邮件标题后缀，如 " (测试)"、" (实时预览)"
}

// FailedSource 抓取失败的源及其原因，用于向用户反馈
type FailedSource struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// Result 一次晨报生成的产物
type Result struct {
	HTML          string         // 渲染完成的完整邮件 HTML
	Title         string         // 邮件标题
	TotalCount    int            // 有效新闻总条数
	SourceCount   int            // 贡献内容的来源数
	CategoryCount int            // 分类板块数
	AIDigest      string         // AI 生成的原始 Markdown 速览（未启用或失败时为空）
	FailedSources []FailedSource // 本次抓取失败的源清单
}

// Build 执行完整的晨报生成流程。
//
// aiClient 传 nil 或未配置 APIKey 时跳过 AI 速览；AI 调用失败仅记录警告并
// 降级为普通晨报，不会导致整体失败。
func Build(cfg *config.Config, aiClient *ai.Client, opts Options) (*Result, error) {
	sources := cfg.EnabledSources()
	if len(sources) == 0 {
		return nil, ErrNoSources
	}
	slog.Info("配置加载完成", "total", len(cfg.Sources), "enabled", len(sources))

	// 1. 并发抓取
	if opts.FetchTimeout <= 0 {
		opts.FetchTimeout = 20 * time.Second
	}
	results := fetcher.New(opts.FetchTimeout).FetchAll(sources)

	var failedSources []FailedSource
	for _, r := range results {
		if r.Err != nil {
			failedSources = append(failedSources, FailedSource{
				Name:   r.Source.Name,
				URL:    r.Source.URL,
				Reason: r.Err.Error(),
			})
		}
	}
	if len(results) == len(failedSources) && len(failedSources) > 0 {
		return nil, fmt.Errorf("%w: 全部 %d 个源均失败", ErrNoResults, len(failedSources))
	}

	// 2. 解析和清洗
	p := parser.New(cfg.Settings.SummaryMaxLength, cfg.Settings.HoursLookback)
	allItems := p.ParseAll(results, cfg.Settings.MaxItemsPerSource)
	if len(allItems) == 0 {
		return nil, ErrNoItems
	}

	// 3. 分组统计
	sections := parser.GroupByCategory(allItems, cfg.Sources)

	sourceSet := make(map[string]struct{})
	for _, item := range allItems {
		sourceSet[item.Source] = struct{}{}
	}

	slog.Info("统计信息",
		"total", len(allItems),
		"sources", len(sourceSet),
		"categories", len(sections),
		"failed_sources", len(failedSources),
	)

	// 4. 可选 AI 要闻速览提炼（失败降级为普通晨报）
	var aiDigest string
	var aiSummaryHTML template.HTML
	if aiClient != nil && aiClient.IsEnabled() {
		digest, aiErr := aiClient.GenerateDailyDigest(context.Background(), allItems)
		if aiErr != nil {
			slog.Warn("AI 速览生成失败，降级为普通晨报", "error", aiErr)
		} else if digest != "" {
			aiDigest = digest
			aiSummaryHTML = renderer.FormatAISummaryToHTML(digest)
		}
	}

	// 5. 渲染
	today := timeutil.TodayString()
	title := fmt.Sprintf("NewsPocket 晨报 - %s%s", today, opts.TitleSuffix)

	htmlContent, err := renderer.New().Render(renderer.TemplateData{
		Title:         title,
		Date:          today,
		TotalCount:    len(allItems),
		SourceCount:   len(sourceSet),
		CategoryCount: len(sections),
		Sections:      sections,
		AISummary:     aiSummaryHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("模板渲染失败: %w", err)
	}

	return &Result{
		HTML:          htmlContent,
		Title:         title,
		TotalCount:    len(allItems),
		SourceCount:   len(sourceSet),
		CategoryCount: len(sections),
		AIDigest:      aiDigest,
		FailedSources: failedSources,
	}, nil
}
