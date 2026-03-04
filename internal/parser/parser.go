// Package parser 负责清洗、过滤和分类新闻条目。
// 移植自 Python 版 ContentParser，保留句子边界智能截断和噪音过滤。
package parser

import (
	"fmt"
	"html"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/fetcher"
)

// NewsItem 最终输出的新闻条目
type NewsItem struct {
	Title    string    `json:"title"`
	Time     string    `json:"time"` // 格式化后的时间字符串
	TimeObj  time.Time `json:"-"`    // 原始时间，用于排序
	Summary  string    `json:"summary"`
	Link     string    `json:"link"`
	Source   string    `json:"source"`
	Category string    `json:"category"`
}

// SourceGroup 按来源分组的新闻
type SourceGroup struct {
	Name      string
	Collapsed bool
	Items     []NewsItem
}

// CategorySection 按分类的新闻板块
type CategorySection struct {
	Index  int
	Name   string
	Groups []SourceGroup
}

// Parser 内容解析器
type Parser struct {
	maxSummaryLength int
	hoursLookback    int
	cutoffTime       time.Time
}

// 正则模式
var (
	htmlTagPattern    = regexp.MustCompile(`<[^>]+>`)
	whitespacePattern = regexp.MustCompile(`\s+`)
	noisePatterns     = []*regexp.Regexp{
		regexp.MustCompile(`\[图片\]`),
		regexp.MustCompile(`\[视频\]`),
		regexp.MustCompile(`点击查看.*`),
		regexp.MustCompile(`阅读原文.*`),
		regexp.MustCompile(`展开全文.*`),
	}
)

// New 创建解析器
func New(maxSummaryLen, hoursLookback int) *Parser {
	return &Parser{
		maxSummaryLength: maxSummaryLen,
		hoursLookback:    hoursLookback,
		cutoffTime:       time.Now().UTC().Add(-time.Duration(hoursLookback) * time.Hour),
	}
}

// CleanHTML 去除 HTML 标签、解码实体、移除噪音
func (p *Parser) CleanHTML(text string) string {
	if text == "" {
		return ""
	}

	// 解码 HTML 实体
	text = html.UnescapeString(text)

	// 去除 HTML 标签
	text = htmlTagPattern.ReplaceAllString(text, " ")

	// 去除噪音内容
	for _, pattern := range noisePatterns {
		text = pattern.ReplaceAllString(text, "")
	}

	// 规范化空白
	text = whitespacePattern.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// TruncateSummary 智能截断摘要（在句子边界截断）
func (p *Parser) TruncateSummary(text string) string {
	runes := []rune(text)
	if len(runes) <= p.maxSummaryLength {
		return text
	}

	truncatedRunes := runes[:p.maxSummaryLength]

	// 在句子边界处截断（至少保留 60%）
	minPos := int(float64(p.maxSummaryLength) * 0.6)
	sentenceEnds := map[rune]bool{
		'。': true, '！': true, '？': true,
		'.': true, '!': true, '?': true,
		'；': true, ';': true,
	}

	bestPos := -1
	for i := len(truncatedRunes) - 1; i >= minPos; i-- {
		if sentenceEnds[truncatedRunes[i]] {
			bestPos = i
			break
		}
	}

	if bestPos > 0 {
		return string(runes[:bestPos+1])
	}

	return strings.TrimSpace(string(truncatedRunes)) + "..."
}

// isRecent 检查时间是否在有效范围内
func (p *Parser) isRecent(t time.Time) bool {
	if t.IsZero() {
		return true // 无时间信息的默认包含
	}
	return t.After(p.cutoffTime)
}

// formatTime 格式化时间为北京时间字符串
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "未知时间"
	}
	beijing := time.FixedZone("CST", 8*3600)
	return t.In(beijing).Format("2006-01-02 15:04")
}

// ParseAll 解析所有抓取结果，返回按时间排序的条目列表
func (p *Parser) ParseAll(results []fetcher.FetchResult, maxPerSource int) []NewsItem {
	var allItems []NewsItem

	for _, result := range results {
		items := p.parseFeedResult(result, maxPerSource)
		allItems = append(allItems, items...)
	}

	// 全局按时间排序（最新在前）
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].TimeObj.After(allItems[j].TimeObj)
	})

	return allItems
}

// parseFeedResult 解析单个源的结果
func (p *Parser) parseFeedResult(result fetcher.FetchResult, maxItems int) (items []NewsItem) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("解析源结果时发生 panic",
				"source", result.Source.Name,
				"error", fmt.Sprintf("%v", r),
			)
		}
	}()

	for _, entry := range result.Entries {
		item := p.parseEntry(entry, result.Source)
		if item != nil {
			items = append(items, *item)
		}
	}

	// 按时间排序后截取
	sort.Slice(items, func(i, j int) bool {
		return items[i].TimeObj.After(items[j].TimeObj)
	})

	if len(items) > maxItems {
		items = items[:maxItems]
	}

	return items
}

// parseEntry 解析单个条目
func (p *Parser) parseEntry(entry fetcher.Entry, source config.Source) *NewsItem {
	isJSONAPI := entry.SourceType == "json_api"

	// 时间过滤（JSON API 放宽限制，可能没有时间字段）
	if !isJSONAPI && !p.isRecent(entry.Published) {
		return nil
	}

	title := p.CleanHTML(entry.Title)
	if title == "" {
		return nil
	}

	summary := p.TruncateSummary(p.CleanHTML(entry.Summary))

	return &NewsItem{
		Title:    title,
		Time:     formatTime(entry.Published),
		TimeObj:  entry.Published,
		Summary:  summary,
		Link:     entry.Link,
		Source:   source.Name,
		Category: source.Category,
	}
}

// 预定义分类顺序
var categoryOrder = []string{"行业动态", "全球热点", "科技生活", "社交热点", "其他"}

// GroupByCategory 按分类分组，返回有序的 CategorySection 列表
func GroupByCategory(items []NewsItem, sources []config.Source) []CategorySection {
	// 构建源名到配置的映射
	sourceMap := make(map[string]config.Source)
	for _, s := range sources {
		sourceMap[s.Name] = s
	}

	// 按分类分组
	grouped := make(map[string][]NewsItem)
	for _, item := range items {
		cat := item.Category
		if cat == "" {
			cat = "其他"
		}
		grouped[cat] = append(grouped[cat], item)
	}

	// 按预定义顺序构建结果
	var sections []CategorySection
	idx := 1

	// 先处理预定义分类
	for _, cat := range categoryOrder {
		catItems, ok := grouped[cat]
		if !ok || len(catItems) == 0 {
			continue
		}
		sections = append(sections, buildSection(idx, cat, catItems, sourceMap))
		idx++
		delete(grouped, cat)
	}

	// 再处理其他未预定义的分类
	for cat, catItems := range grouped {
		if len(catItems) == 0 {
			continue
		}
		sections = append(sections, buildSection(idx, cat, catItems, sourceMap))
		idx++
	}

	return sections
}

// buildSection 构建单个分类板块
func buildSection(idx int, name string, items []NewsItem, sourceMap map[string]config.Source) CategorySection {
	// 按来源分组
	sourceGroups := make(map[string][]NewsItem)
	var sourceOrder []string

	for _, item := range items {
		if _, exists := sourceGroups[item.Source]; !exists {
			sourceOrder = append(sourceOrder, item.Source)
		}
		sourceGroups[item.Source] = append(sourceGroups[item.Source], item)
	}

	var groups []SourceGroup
	for _, sourceName := range sourceOrder {
		sc, ok := sourceMap[sourceName]
		collapsed := false
		if ok {
			collapsed = sc.Collapsed
		}

		groups = append(groups, SourceGroup{
			Name:      sourceName,
			Collapsed: collapsed,
			Items:     sourceGroups[sourceName],
		})
	}

	return CategorySection{
		Index:  idx,
		Name:   name,
		Groups: groups,
	}
}
