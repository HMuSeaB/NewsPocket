package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

// JevReranker 基于 Jev 的智能精选与重排序器
type JevReranker struct {
	client     *JevClient
	minScore   float64 // 最低入选分数（低于此分视为水文/低价值过滤）
	consoleURL string  // Jev Decision Console 本地推送地址（如 http://localhost:3000）
}

// NewJevReranker 创建重排序器
func NewJevReranker(client *JevClient) *JevReranker {
	return &JevReranker{
		client:     client,
		minScore:   2.5,
		consoleURL: "http://localhost:3000",
	}
}

// RerankAndFilter 对全部新闻进行 Jev 批量打分、智能降噪、分类与 TopN 提取
func (r *JevReranker) RerankAndFilter(
	ctx context.Context,
	items []parser.NewsItem,
	topN int,
	userPreferences string,
) ([]parser.NewsItem, error) {
	if r.client == nil || !r.client.IsEnabled() || len(items) == 0 {
		return items, nil
	}

	if userPreferences == "" {
		userPreferences = "重点挑选高含金量的开发者工具、开源发布、系统底层架构、重大突破与真实可用实践，严格剔除公关软文与同质化新闻"
	}

	// 1. 分批打分 (避免单次请求超过 Token 上限，每批 12 条)
	batchSize := 12
	var scoredItems []parser.NewsItem

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]

		scoredBatch, err := r.client.BatchScoreAndCategorize(ctx, batch, userPreferences)
		if err != nil {
			slog.Warn("批次评分异常，保留原样", "batch_start", i, "err", err)
			scoredItems = append(scoredItems, batch...)
		} else {
			scoredItems = append(scoredItems, scoredBatch...)
		}
	}

	// 2. 排序：按 Jev Score 从高到低降序排列
	sort.SliceStable(scoredItems, func(i, j int) bool {
		return scoredItems[i].Score > scoredItems[j].Score
	})

	// 3. 智能过滤：剔除低于阈值的水文
	var highQualityItems []parser.NewsItem
	for _, item := range scoredItems {
		if item.Score >= r.minScore {
			highQualityItems = append(highQualityItems, item)
		}
	}

	// 保底逻辑：如果阈值过滤后太少，则至少保留前 topN 条最高分的
	if len(highQualityItems) < topN && len(scoredItems) > 0 {
		limit := topN
		if limit > len(scoredItems) {
			limit = len(scoredItems)
		}
		highQualityItems = scoredItems[:limit]
	} else if len(highQualityItems) > topN && topN > 0 {
		highQualityItems = highQualityItems[:topN]
	}

	slog.Info("Jev 智能精选完成",
		"original_count", len(items),
		"filtered_count", len(highQualityItems),
		"top_score", highQualityItems[0].Score,
	)

	// 4. 异步推送决策至 Jev Decision Console (若控制台运行中)
	go r.reportToDecisionConsole(len(items), highQualityItems, userPreferences)

	return highQualityItems, nil
}

// reportToDecisionConsole 向本地运行的 Jev Decision Console 推送本次精选决策
func (r *JevReranker) reportToDecisionConsole(
	totalCount int,
	selected []parser.NewsItem,
	prefs string,
) {
	if r.consoleURL == "" || len(selected) == 0 {
		return
	}

	type candidateOption struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		RiskLevel   string `json:"riskLevel"`
	}

	options := make([]candidateOption, 0, len(selected))
	for i, item := range selected {
		if i >= 5 {
			break // 控制台展示前 5 个高分候选
		}
		options = append(options, candidateOption{
			ID:          item.Link,
			Name:        item.Title,
			Description: item.Summary,
			RiskLevel:   "low",
		})
	}

	payload := map[string]any{
		"title":       "NewsPocket: 今日全网资讯 Jev 智能精选与降噪",
		"description": "基于 TypeSafe Jev 认知模型，从海量抓取流中智能剔除营销水文，选出最高价值条目",
		"category":    "refactor",
		"state": map[string]any{
			"project":          "NewsPocket",
			"total_fetched":    totalCount,
			"selected_count":   len(selected),
			"user_preferences": prefs,
		},
		"options": options,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 2 * time.Second}
	_, _ = client.Post(r.consoleURL+"/api/decisions", "application/json", bytes.NewReader(jsonBytes))
}
