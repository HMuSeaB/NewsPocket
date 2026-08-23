// Package ai 提供基于大语言模型（LLM）的每日要闻智能速览生成能力。
// 兼容 OpenAI / DeepSeek / Moonshot / Ollama 等标准 Chat Completions API 协议。
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
	"strconv"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

// Config 大模型 API 相关配置
type Config struct {
	APIKey     string        `json:"api_key"`
	BaseURL    string        `json:"base_url"`
	Model      string        `json:"model"`
	Prompt     string        `json:"prompt"`
	Timeout    time.Duration `json:"timeout"`
	MaxItems   int           `json:"max_items"` // 送入 AI 提炼的最大条目数
}

// Client 大模型客户端
type Client struct {
	cfg      Config
	client   *http.Client
	disabled bool // settings.ai.enabled 显式停用标记
}

// DefaultSystemPrompt 默认系统 Prompt
const DefaultSystemPrompt = `你是一位专业、客观的新闻与行业趋势主编。
请根据提供的今日精选资讯列表，提炼一份高信息密度的「今日要闻 1 分钟速读」。
要求：
1. 提炼 3~5 个最重要的核心要点或重大动态，每条使用序号 (1., 2., 3...) 列出。
2. 每条格式为：**【核心主题/事件】** 具体动态与深远影响简析（1~2 句话，拒绝冗余套话）。
3. 语言精炼有力，直接输出要点列表，严禁输出任何开场白、问候语或总结收尾语。`

// LoadConfigFromEnv 从环境变量加载 AI 相关配置
func LoadConfigFromEnv() Config {
	apiKey := strings.TrimSpace(os.Getenv("AI_API_KEY"))
	baseURL := normalizeBaseURL(os.Getenv("AI_BASE_URL"))

	model := strings.TrimSpace(os.Getenv("AI_MODEL"))
	if model == "" {
		model = "deepseek-chat"
	}

	prompt := strings.TrimSpace(os.Getenv("AI_PROMPT"))
	if prompt == "" {
		prompt = DefaultSystemPrompt
	}

	timeoutSec := 45
	if tStr := os.Getenv("AI_TIMEOUT"); tStr != "" {
		if tVal, err := strconv.Atoi(tStr); err == nil && tVal > 0 {
			timeoutSec = tVal
		}
	}

	maxItems := 25
	if mStr := os.Getenv("AI_MAX_ITEMS"); mStr != "" {
		if mVal, err := strconv.Atoi(mStr); err == nil && mVal > 0 {
			maxItems = mVal
		}
	}

	return Config{
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    model,
		Prompt:   prompt,
		Timeout:  time.Duration(timeoutSec) * time.Second,
		MaxItems: maxItems,
	}
}

// normalizeBaseURL 规范化 API BaseURL（去尾部斜杠与多余的 /chat/completions）
func normalizeBaseURL(raw string) string {
	baseURL := strings.TrimSpace(raw)
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return strings.TrimSuffix(baseURL, "/chat/completions")
}

// LoadConfigMerged 合并 sources.json 的 settings.ai 与环境变量生成最终配置。
// 优先级：settings.ai 显式值 > 环境变量 > 内置默认值，
// 因此 GitHub Actions 用户可以只配 Secrets，桌面 GUI 用户则直接写 JSON。
func LoadConfigMerged(s *config.AISettings) Config {
	cfg := LoadConfigFromEnv()
	if s == nil {
		return cfg
	}

	if s.APIKey != "" {
		cfg.APIKey = strings.TrimSpace(s.APIKey)
	}
	if s.BaseURL != "" {
		cfg.BaseURL = normalizeBaseURL(s.BaseURL)
	}
	if s.Model != "" {
		cfg.Model = strings.TrimSpace(s.Model)
	}
	if s.Prompt != "" {
		cfg.Prompt = strings.TrimSpace(s.Prompt)
	}
	if s.TimeoutSeconds > 0 {
		cfg.Timeout = time.Duration(s.TimeoutSeconds) * time.Second
	}
	if s.MaxItems > 0 {
		cfg.MaxItems = s.MaxItems
	}

	return cfg
}

// NewClient 创建 AI 客户端实例（仅读取环境变量配置）
func NewClient() *Client {
	return NewClientWithConfig(LoadConfigFromEnv())
}

// NewClientFromSettings 使用「sources.json settings.ai + 环境变量」合并配置创建客户端，
// 并尊重 settings.ai 的 enabled 开关。CLI 与 GUI 均应使用此构造方式。
func NewClientFromSettings(s *config.AISettings) *Client {
	c := NewClientWithConfig(LoadConfigMerged(s))
	c.disabled = !s.IsEnabled()
	return c
}

// NewClientWithConfig 使用指定配置创建客户端
func NewClientWithConfig(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 45 * time.Second
	}
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = 25
	}
	if cfg.Prompt == "" {
		cfg.Prompt = DefaultSystemPrompt
	}

	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// IsEnabled 检查 AI 模块是否已启用且配置了 APIKey
func (c *Client) IsEnabled() bool {
	return !c.disabled && c.cfg.APIKey != ""
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatCompletionChoice struct {
	Message chatMessage `json:"message"`
}

type chatCompletionResponse struct {
	Choices []chatCompletionChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateDailyDigest 生成今日新闻速读总结
func (c *Client) GenerateDailyDigest(ctx context.Context, items []parser.NewsItem) (string, error) {
	if !c.IsEnabled() {
		return "", nil
	}

	if len(items) == 0 {
		return "", nil
	}

	slog.Info("开始调用大模型生成今日要闻速览",
		"model", c.cfg.Model,
		"items_count", len(items),
		"base_url", c.cfg.BaseURL,
	)

	// 1. 组装送入大模型的新闻上下文
	limit := len(items)
	if limit > c.cfg.MaxItems {
		limit = c.cfg.MaxItems
	}

	var sb strings.Builder
	sb.WriteString("以下是今日抓取到的精选资讯列表：\n\n")
	for i := 0; i < limit; i++ {
		item := items[i]
		sb.WriteString(fmt.Sprintf("%d. 【%s】%s", i+1, item.Source, item.Title))
		if item.Summary != "" {
			sb.WriteString(" —— " + item.Summary)
		}
		sb.WriteString("\n")
	}

	// 2. 构造请求 Payload
	reqPayload := chatCompletionRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: c.cfg.Prompt},
			{Role: "user", Content: sb.String()},
		},
		Temperature: 0.3,
		MaxTokens:   1000,
	}

	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("序列化大模型请求失败: %w", err)
	}

	endpoint := c.cfg.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	// 3. 发起请求
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求大模型接口失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取大模型响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("大模型接口返回异常状态码 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("解析大模型响应 JSON 失败: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("大模型返回业务错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("大模型返回内容为空")
	}

	digest := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	slog.Info("今日要闻速览生成成功", "length", len(digest))

	return digest, nil
}
