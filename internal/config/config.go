// Package config 负责加载和管理 sources.json 配置文件。
// 结构体设计完全兼容现有 Python 版本的 JSON 格式，确保零迁移成本。
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 顶层配置结构，对应 sources.json 根对象
type Config struct {
	Sources  []Source `json:"sources"`
	Settings Settings `json:"settings"`
}

// Source 单个新闻源配置
type Source struct {
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	Category  string            `json:"category"`
	Type      string            `json:"type"`    // "rss" | "json_api" | "script"
	Enabled   *bool             `json:"enabled"` // 指针类型，区分 false 和未设置（默认 true）
	Collapsed bool              `json:"collapsed"`
	Comment   string            `json:"comment,omitempty"`
	Method    string            `json:"method,omitempty"` // GET/POST, 仅 json_api
	Headers   map[string]string `json:"headers,omitempty"`
	Body      json.RawMessage   `json:"body,omitempty"` // POST body, 保留原始 JSON

	// JSON API 专用
	JSONConfig *JSONConfig `json:"json_config,omitempty"`
}

// IsEnabled 返回源是否启用。未设置 enabled 字段时默认为 true。
func (s *Source) IsEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// JSONConfig JSON API 源的字段映射配置
type JSONConfig struct {
	ItemsPath    string `json:"items_path"`
	TitleField   string `json:"title_field"`
	LinkField    string `json:"link_field,omitempty"`
	LinkTemplate string `json:"link_template,omitempty"`
	SummaryField string `json:"summary_field,omitempty"`
	TimeField    string `json:"time_field,omitempty"`
	// TimeZone 解析无时区时间字符串时使用的时区。
	// 支持 IANA 名称 ("Asia/Shanghai") 或小时偏移 ("+8")；
	// 缺省按北京时间 (UTC+8) 处理，适配中文源接口。
	TimeZone string `json:"time_zone,omitempty"`
}

// AISettings 大模型「今日要闻速读」配置，位于 sources.json 的 settings.ai 节。
// 所有字段均可留空 —— 留空时自动回退读取对应环境变量
// (AI_API_KEY / AI_BASE_URL / AI_MODEL / AI_PROMPT / AI_TIMEOUT / AI_MAX_ITEMS)，
// 因此 GitHub Actions 用户可以继续使用 Secrets 而无需改动此节。
//
// ⚠️ 若在公开仓库中使用，请勿将 api_key 写入本文件，应使用环境变量/Secrets 注入。
type AISettings struct {
	Enabled *bool `json:"enabled,omitempty"` // 显式 false 可整体停用 AI 模块；缺省视为 true

	APIKey         string `json:"api_key,omitempty"`
	BaseURL        string `json:"base_url,omitempty"`    // 兼容 OpenAI 协议，如 https://api.deepseek.com/v1
	Model          string `json:"model,omitempty"`       // 如 deepseek-chat、gpt-4o-mini
	Prompt         string `json:"prompt,omitempty"`      // 自定义系统提示词
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
	MaxItems       int    `json:"max_items,omitempty"`   // 送入大模型提炼的最大条目数
}

// IsEnabled 返回 AI 模块是否被显式停用。未设置时默认为 true。
func (a *AISettings) IsEnabled() bool {
	if a == nil || a.Enabled == nil {
		return true
	}
	return *a.Enabled
}

// Settings 全局设置
type Settings struct {
	MaxItemsPerSource int         `json:"max_items_per_source"`
	HoursLookback     int         `json:"hours_lookback"`
	SummaryMaxLength  int         `json:"summary_max_length"`
	AI                *AISettings `json:"ai,omitempty"`
}

// LoadConfig 从指定路径加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 填充默认值
	if cfg.Settings.MaxItemsPerSource == 0 {
		cfg.Settings.MaxItemsPerSource = 5
	}
	if cfg.Settings.HoursLookback == 0 {
		cfg.Settings.HoursLookback = 24
	}
	if cfg.Settings.SummaryMaxLength == 0 {
		cfg.Settings.SummaryMaxLength = 200
	}

	return &cfg, nil
}

// EnabledSources 返回所有启用的源
func (c *Config) EnabledSources() []Source {
	var result []Source
	for _, s := range c.Sources {
		if s.IsEnabled() {
			result = append(result, s)
		}
	}
	return result
}
