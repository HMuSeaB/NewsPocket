package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

// JevClient 是 TypeSafe System One (Jev) 认知模型客户端
type JevClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// JevQuestion 表示单个强类型认知问题 (Score, Choice, Noul)
type JevQuestion struct {
	Type         string      `json:"type"`
	Instructions any         `json:"instructions"`
	Criteria     any         `json:"criteria,omitempty"`
}

// JevSystemOneRequest 请求体
type JevSystemOneRequest struct {
	State     any                    `json:"state"`
	Model     string                 `json:"model"`
	Questions map[string]JevQuestion `json:"questions"`
}

// JevAnswer 返回判定结果
type JevAnswer struct {
	Type          string             `json:"type"`
	Value         any                `json:"value,omitempty"`
	Confidence    float64            `json:"confidence,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Probability   float64            `json:"probability,omitempty"`
}

// JevSystemOneResponse 响应体
type JevSystemOneResponse struct {
	Answers map[string]JevAnswer `json:"answers"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewJevClient 创建 Jev 客户端实例
func NewJevClient(apiKey, baseURL string) *JevClient {
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY"))
	}
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("TYPESAFE_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = "https://api.typesafe.ai/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &JevClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   "jev-latest",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsEnabled 检查 Jev 是否配置可用
func (j *JevClient) IsEnabled() bool {
	return j.apiKey != ""
}

// BatchScoreAndCategorize 运用推测式展开 (Speculative Fan-out)，单次请求对一组新闻进行并发评分与分类
func (j *JevClient) BatchScoreAndCategorize(
	ctx context.Context,
	items []parser.NewsItem,
	userPreferences string,
) ([]parser.NewsItem, error) {
	if !j.IsEnabled() || len(items) == 0 {
		return items, nil
	}

	slog.Info("开始调用 Jev System One 进行批量智能打分与专栏聚类 (Speculative Fan-out)",
		"batch_size", len(items),
		"model", j.model,
	)

	// 1. 构造共享 State (包含新闻列表与用户偏好)
	type newsContext struct {
		Index   int    `json:"index"`
		Title   string `json:"title"`
		Source  string `json:"source"`
		Summary string `json:"summary"`
	}

	stateNews := make([]newsContext, len(items))
	for i, item := range items {
		stateNews[i] = newsContext{
			Index:   i,
			Title:   item.Title,
			Source:  item.Source,
			Summary: item.Summary,
		}
	}

	state := map[string]any{
		"user_preferences": userPreferences,
		"news_batch":       stateNews,
	}

	// 2. 针对每篇新闻并行提问：评分 (Score) + 分类 (Choice)
	questions := make(map[string]JevQuestion)
	for i, item := range items {
		// 价值评分原语
		scoreKey := fmt.Sprintf("score_%d", i)
		questions[scoreKey] = JevQuestion{
			Type: "score",
			Instructions: fmt.Sprintf(
				"评估第 %d 条新闻【%s】对高价值技术开发者的信息密度、深度与实用突破性（0~5分）。剔除营销水文。",
				i, item.Title,
			),
			Criteria: map[string]string{
				"0": "毫无价值的营销软文、标题党、广告或同质化低质资讯",
				"3": "有一定参考价值的常规行业动态、技术更新或实用工具介绍",
				"5": "重磅技术突破、革命性开源发布、深度架构复盘或关键安全漏洞",
			},
		}

		// 专栏分类原语
		catKey := fmt.Sprintf("cat_%d", i)
		questions[catKey] = JevQuestion{
			Type: "choice",
			Instructions: fmt.Sprintf("为第 %d 条新闻【%s】选择最契合的技术板块分类。", i, item.Title),
			Criteria: map[string]string{
				"🛠️ 开发者工具 / 开源项目": "关于实用开发工具、开源框架发布、CLI/IDE 插件等",
				"🔬 AI 与前沿架构":       "关于大模型算法、系统架构演进、前沿技术论文与推理引擎",
				"💼 行业洞察 / 商业观察":   "科技巨头动态、投融资、行业政策与战略趋势",
				"⚡ 突发快讯 / 安全运维":   "严重安全漏洞披露、宕机事件、突发基础设施告警",
			},
		}
	}

	// 3. 构造并发送 HTTP 请求
	reqBody := JevSystemOneRequest{
		State:     state,
		Model:     j.model,
		Questions: questions,
	}

	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		return items, fmt.Errorf("序列化 Jev 请求失败: %w", err)
	}

	endpoint := j.baseURL + "/systemone"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return items, fmt.Errorf("创建 Jev HTTP 请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+j.apiKey)

	resp, err := j.client.Do(httpReq)
	if err != nil {
		slog.Warn("Jev 服务调用失败，自动降级为原始顺序", "err", err)
		return items, nil
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("读取 Jev 响应失败，自动降级", "err", err)
		return items, nil
	}

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Jev 接口返回异常状态码，自动降级", "status", resp.StatusCode, "body", string(respBytes))
		return items, nil
	}

	var jevResp JevSystemOneResponse
	if err := json.Unmarshal(respBytes, &jevResp); err != nil {
		slog.Warn("解析 Jev 响应失败，自动降级", "err", err)
		return items, nil
	}

	// 4. 解析各问题答案并回填至 NewsItem
	resultItems := make([]parser.NewsItem, len(items))
	copy(resultItems, items)

	for i := range resultItems {
		scoreKey := fmt.Sprintf("score_%d", i)
		if ans, ok := jevResp.Answers[scoreKey]; ok {
			// TypeSafe Score 返回数值
			if val, ok := ans.Value.(float64); ok {
				resultItems[i].Score = val
			} else if ans.Probability > 0 {
				resultItems[i].Score = ans.Probability * 5.0
			} else {
				resultItems[i].Score = 3.0 // 默认中位分
			}
		} else {
			resultItems[i].Score = 3.0
		}

		catKey := fmt.Sprintf("cat_%d", i)
		if ans, ok := jevResp.Answers[catKey]; ok {
			if val, ok := ans.Value.(string); ok && val != "" {
				resultItems[i].JevCategory = val
			}
		}
	}

	slog.Info("Jev 批量评分与分类完成", "total_scored", len(resultItems))
	return resultItems, nil
}
