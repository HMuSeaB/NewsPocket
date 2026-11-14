// Package pusher 提供多渠道消息推送功能，
// 支持 企业微信群机器人、飞书自定义机器人、钉钉机器人、Telegram Bot 及通用 Webhook。
package pusher

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
)

// ChannelType 推送渠道类型
type ChannelType string

const (
	ChannelFeishu   ChannelType = "feishu"
	ChannelWeCom    ChannelType = "wecom"
	ChannelDingTalk ChannelType = "dingtalk"
	ChannelTelegram ChannelType = "telegram"
	ChannelWebhook  ChannelType = "webhook"
)

// Config 单个推送渠道配置
type Config struct {
	Type     ChannelType `json:"type"`
	URL      string      `json:"url"`
	Token    string      `json:"token,omitempty"`    // 针对 Telegram
	ChatID   string      `json:"chat_id,omitempty"`  // 针对 Telegram
	Disabled bool        `json:"disabled,omitempty"`
}

// Payload 推送的消息体数据
type Payload struct {
	Title      string `json:"title"`
	Markdown   string `json:"markdown"`
	TotalCount int    `json:"total_count"`
	AIDigest   string `json:"ai_digest,omitempty"`
}

// Pusher 多通道推送客户端
type Pusher struct {
	client  *http.Client
	configs []Config
}

// New 创建推送管理器
func New(configs []Config) *Pusher {
	return &Pusher{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		configs: configs,
	}
}

// LoadFromEnv 从环境变量自动发现启用的推送渠道
func LoadFromEnv() []Config {
	var configs []Config

	if url := os.Getenv("WECOM_WEBHOOK"); url != "" {
		configs = append(configs, Config{Type: ChannelWeCom, URL: url})
	}
	if url := os.Getenv("FEISHU_WEBHOOK"); url != "" {
		configs = append(configs, Config{Type: ChannelFeishu, URL: url})
	}
	if url := os.Getenv("DINGTALK_WEBHOOK"); url != "" {
		configs = append(configs, Config{Type: ChannelDingTalk, URL: url})
	}
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		chatID := os.Getenv("TELEGRAM_CHAT_ID")
		if chatID != "" {
			configs = append(configs, Config{
				Type:   ChannelTelegram,
				Token:  token,
				ChatID: chatID,
			})
		}
	}
	if url := os.Getenv("WEBHOOK_URL"); url != "" {
		configs = append(configs, Config{Type: ChannelWebhook, URL: url})
	}

	return configs
}

// HasActiveChannels 判断是否有可用的推送通道
func (p *Pusher) HasActiveChannels() bool {
	for _, c := range p.configs {
		if !c.Disabled {
			return true
		}
	}
	return false
}

// PushAll 向所有配置的通道并发推送消息
func (p *Pusher) PushAll(ctx context.Context, payload Payload) map[ChannelType]error {
	results := make(map[ChannelType]error)

	for _, cfg := range p.configs {
		if cfg.Disabled {
			continue
		}

		var err error
		switch cfg.Type {
		case ChannelWeCom:
			err = p.sendWeCom(ctx, cfg.URL, payload)
		case ChannelFeishu:
			err = p.sendFeishu(ctx, cfg.URL, payload)
		case ChannelDingTalk:
			err = p.sendDingTalk(ctx, cfg.URL, payload)
		case ChannelTelegram:
			err = p.sendTelegram(ctx, cfg.Token, cfg.ChatID, payload)
		case ChannelWebhook:
			err = p.sendGenericWebhook(ctx, cfg.URL, payload)
		default:
			err = fmt.Errorf("不支持的推送通道类型: %s", cfg.Type)
		}

		results[cfg.Type] = err
		if err != nil {
			slog.Error("渠道推送失败", "channel", cfg.Type, "error", err)
		} else {
			slog.Info("渠道推送成功", "channel", cfg.Type)
		}
	}

	return results
}

// sendWeCom 推送至企业微信机器人
func (p *Pusher) sendWeCom(ctx context.Context, webhookURL string, payload Payload) error {
	// 企业微信 Markdown 限制单条 4096 字节
	content := payload.Markdown
	if len(content) > 3800 {
		content = content[:3800] + "\n\n...(内容过长已截断，完整请查看邮件)"
	}

	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	return p.postJSON(ctx, webhookURL, body)
}

// sendFeishu 推送至飞书机器人
func (p *Pusher) sendFeishu(ctx context.Context, webhookURL string, payload Payload) error {
	// 飞书富文本 post 格式
	lines := strings.Split(payload.Markdown, "\n")
	var elements []map[string]interface{}
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		elements = append(elements, map[string]interface{}{
			"tag":  "text",
			"text": l + "\n",
		})
	}

	body := map[string]interface{}{
		"msg_type": "post",
		"content": map[string]interface{}{
			"post": map[string]interface{}{
				"zh_cn": map[string]interface{}{
					"title":   payload.Title,
					"content": [][]map[string]interface{}{elements},
				},
			},
		},
	}
	return p.postJSON(ctx, webhookURL, body)
}

// sendDingTalk 推送至钉钉机器人
func (p *Pusher) sendDingTalk(ctx context.Context, webhookURL string, payload Payload) error {
	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": payload.Title,
			"text":  payload.Markdown,
		},
	}
	return p.postJSON(ctx, webhookURL, body)
}

// sendTelegram 推送至 Telegram Bot
func (p *Pusher) sendTelegram(ctx context.Context, token, chatID string, payload Payload) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	text := payload.Markdown
	// Telegram 限制 4096 字符
	if len(text) > 4000 {
		text = text[:4000] + "\n\n...(truncated)"
	}

	body := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	return p.postJSON(ctx, apiURL, body)
}

// sendGenericWebhook 通用 Webhook 推送
func (p *Pusher) sendGenericWebhook(ctx context.Context, webhookURL string, payload Payload) error {
	return p.postJSON(ctx, webhookURL, payload)
}

// postJSON 执行标准 HTTP POST 发送 JSON
func (p *Pusher) postJSON(ctx context.Context, rawURL string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("接口返回异常状态码 %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
