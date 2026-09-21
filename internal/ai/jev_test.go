package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

func TestJevClient_BatchScoreAndCategorize(t *testing.T) {
	// 启动 Mock HTTP 服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		resp := JevSystemOneResponse{
			Answers: map[string]JevAnswer{
				"score_0": {
					Type:  "score",
					Value: 4.8,
				},
				"cat_0": {
					Type:  "choice",
					Value: "🛠️ 开发者工具 / 开源项目",
				},
				"score_1": {
					Type:  "score",
					Value: 1.2,
				},
				"cat_1": {
					Type:  "choice",
					Value: "💼 行业洞察 / 商业观察",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	client := NewJevClient("test-key", mockServer.URL)
	items := []parser.NewsItem{
		{
			Title:   "Jev 开源！System One 架构重塑 Agent 开发范式",
			Source:  "Hacker News",
			Summary: "TypeSafe 推出 Jev，专注低延迟强类型离散决策",
		},
		{
			Title:   "某公司发布毫无新意的软文广告",
			Source:  "科技快讯",
			Summary: "购买我们的服务享受九折优惠",
		},
	}

	reranker := NewJevReranker(client)
	filtered, err := reranker.RerankAndFilter(context.Background(), items, 1, "")
	if err != nil {
		t.Fatalf("RerankAndFilter failed: %v", err)
	}

	if len(filtered) != 1 {
		t.Fatalf("Expected 1 item after filtering, got %d", len(filtered))
	}

	if filtered[0].Score != 4.8 {
		t.Errorf("Expected score 4.8, got %f", filtered[0].Score)
	}

	if filtered[0].JevCategory != "🛠️ 开发者工具 / 开源项目" {
		t.Errorf("Expected category '🛠️ 开发者工具 / 开源项目', got '%s'", filtered[0].JevCategory)
	}
}
