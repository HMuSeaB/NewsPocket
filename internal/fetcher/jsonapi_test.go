package fetcher

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
)

func TestFetchJSONAPIPreservesExplicitPostMethodWithoutBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if len(body) != 0 {
			t.Fatalf("expected empty body, got %q", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"title":"hello"}]}`))
	}))
	defer server.Close()

	result, err := fetchJSONAPI(context.Background(), config.Source{
		Name:   "post-no-body",
		URL:    server.URL,
		Type:   "json_api",
		Method: http.MethodPost,
		JSONConfig: &config.JSONConfig{
			ItemsPath:  "items",
			TitleField: "title",
		},
	}, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("fetchJSONAPI returned error: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
}

func TestParseTimeStringSupportsMilliseconds(t *testing.T) {
	// 13位毫秒时间戳测试 (2024-05-29 09:55:00 UTC)
	msecTime := parseTimeString("1716976500000")
	expectedMsec := time.Date(2024, 5, 29, 9, 55, 0, 0, time.UTC)
	if !msecTime.Equal(expectedMsec) {
		t.Errorf("expected %v, got %v for milliseconds", expectedMsec, msecTime)
	}

	// 10位秒时间戳测试 (2024-05-29 09:55:00 UTC)
	secTime := parseTimeString("1716976500")
	expectedSec := time.Date(2024, 5, 29, 9, 55, 0, 0, time.UTC)
	if !secTime.Equal(expectedSec) {
		t.Errorf("expected %v, got %v for seconds", expectedSec, secTime)
	}

	// 常见日期字符串测试
	strTime := parseTimeString("2026-05-30 12:15:06")
	expectedStr := time.Date(2026, 5, 30, 12, 15, 6, 0, time.UTC)
	if !strTime.Equal(expectedStr) {
		t.Errorf("expected %v, got %v for string format", expectedStr, strTime)
	}
}

func TestBuildLinkSmartURLEscape(t *testing.T) {
	item := map[string]any{
		"word": "苹果 发布会", // 包含空格
		"path": "tech/news",  // 包含斜杠
	}

	// 1. 占位符在 Query 部分（问号之后），应当使用 QueryEscape，空格被转义为 %20 或 +
	templateQuery := "https://example.com/search?q={word}"
	resultQuery := buildLink(item, "", templateQuery)
	expectedQuery := "https://example.com/search?q=" + url.QueryEscape(item["word"].(string))
	if resultQuery != expectedQuery {
		t.Errorf("expected %q, got %q for QueryEscape", expectedQuery, resultQuery)
	}

	// 2. 占位符在 Path 部分（问号之前），应当使用 PathEscape，斜杠被转义为 %2F，空格被转义为 %20
	templatePath := "https://example.com/category/{path}/info"
	resultPath := buildLink(item, "", templatePath)
	expectedPath := "https://example.com/category/" + url.PathEscape(item["path"].(string)) + "/info"
	if resultPath != expectedPath {
		t.Errorf("expected %q, got %q for PathEscape", expectedPath, resultPath)
	}
}
