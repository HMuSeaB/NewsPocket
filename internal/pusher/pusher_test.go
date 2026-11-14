package pusher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPusherAllChannels(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	payload := Payload{
		Title:      "测试早报",
		Markdown:   "# 晨报内容\n- 示例新闻",
		TotalCount: 1,
		AIDigest:   "AI 速览测试",
	}

	t.Run("企业微信推送", func(t *testing.T) {
		p := New([]Config{{Type: ChannelWeCom, URL: server.URL}})
		results := p.PushAll(context.Background(), payload)
		if err := results[ChannelWeCom]; err != nil {
			t.Fatalf("企业微信推送失败: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(receivedBody, &parsed); err != nil {
			t.Fatalf("解析接收到的请求体失败: %v", err)
		}
		if parsed["msgtype"] != "markdown" {
			t.Errorf("msgtype 应为 markdown，实际为: %v", parsed["msgtype"])
		}
	})

	t.Run("飞书推送", func(t *testing.T) {
		p := New([]Config{{Type: ChannelFeishu, URL: server.URL}})
		results := p.PushAll(context.Background(), payload)
		if err := results[ChannelFeishu]; err != nil {
			t.Fatalf("飞书推送失败: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(receivedBody, &parsed); err != nil {
			t.Fatalf("解析接收到的请求体失败: %v", err)
		}
		if parsed["msg_type"] != "post" {
			t.Errorf("msg_type 应为 post，实际为: %v", parsed["msg_type"])
		}
	})

	t.Run("钉钉推送", func(t *testing.T) {
		p := New([]Config{{Type: ChannelDingTalk, URL: server.URL}})
		results := p.PushAll(context.Background(), payload)
		if err := results[ChannelDingTalk]; err != nil {
			t.Fatalf("钉钉推送失败: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(receivedBody, &parsed); err != nil {
			t.Fatalf("解析接收到的请求体失败: %v", err)
		}
		if parsed["msgtype"] != "markdown" {
			t.Errorf("msgtype 应为 markdown，实际为: %v", parsed["msgtype"])
		}
	})

	t.Run("通用 Webhook 推送", func(t *testing.T) {
		p := New([]Config{{Type: ChannelWebhook, URL: server.URL}})
		results := p.PushAll(context.Background(), payload)
		if err := results[ChannelWebhook]; err != nil {
			t.Fatalf("Webhook 推送失败: %v", err)
		}

		if !strings.Contains(receivedContentType, "application/json") {
			t.Errorf("Content-Type 应为 application/json")
		}
	})
}
