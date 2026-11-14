package parser

import (
	"testing"
)

func TestCleanURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空 URL",
			input:    "",
			expected: "",
		},
		{
			name:     "无参数 URL",
			input:    "https://example.com/article/1",
			expected: "https://example.com/article/1",
		},
		{
			name:     "标准 UTM 跟踪参数清洗",
			input:    "https://example.com/news?id=100&utm_source=rss&utm_medium=feed&utm_campaign=daily",
			expected: "https://example.com/news?id=100",
		},
		{
			name:     "B站与微博跟踪参数清洗",
			input:    "https://www.bilibili.com/video/BV1xx?spm_id_from=333.1007&from_source=weibo",
			expected: "https://www.bilibili.com/video/BV1xx",
		},
		{
			name:     "全大写 UTM 参数清洗",
			input:    "https://example.com/post?UTM_SOURCE=twitter&UTM_MEDIUM=social",
			expected: "https://example.com/post",
		},
		{
			name:     "保留合法业务参数",
			input:    "https://s.weibo.com/weibo?q=%23DeepSeek%23&refer_flag=100",
			expected: "https://s.weibo.com/weibo?q=%23DeepSeek%23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanURL(tt.input)
			if got != tt.expected {
				t.Errorf("CleanURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
