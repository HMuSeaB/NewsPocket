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
}

// Settings 全局设置
type Settings struct {
	MaxItemsPerSource int `json:"max_items_per_source"`
	HoursLookback     int `json:"hours_lookback"`
	SummaryMaxLength  int `json:"summary_max_length"`
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
