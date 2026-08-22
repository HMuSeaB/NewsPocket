package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

func TestClientDisabledWhenNoAPIKey(t *testing.T) {
	t.Parallel()

	client := NewClientWithConfig(Config{
		APIKey: "",
	})

	if client.IsEnabled() {
		t.Fatal("expected client to be disabled when APIKey is empty")
	}

	digest, err := client.GenerateDailyDigest(context.Background(), []parser.NewsItem{
		{Title: "Test", Source: "RSS"},
	})
	if err != nil {
		t.Fatalf("expected nil error on disabled client, got: %v", err)
	}
	if digest != "" {
		t.Fatalf("expected empty digest on disabled client, got: %q", digest)
	}
}

func TestGenerateDailyDigestSuccess(t *testing.T) {
	t.Parallel()

	expectedResponse := "1. **【重大发布】** Go 1.24 正式发布，带来更强性能。\n2. **【开源动态】** NewsPocket 升级 AI 要闻速览。"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}

		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if req.Model != "mock-model" {
			t.Errorf("expected model 'mock-model', got %s", req.Model)
		}

		resp := chatCompletionResponse{
			Choices: []chatCompletionChoice{
				{
					Message: chatMessage{
						Role:    "assistant",
						Content: expectedResponse,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithConfig(Config{
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "mock-model",
		Timeout:  5 * time.Second,
		MaxItems: 10,
	})

	items := []parser.NewsItem{
		{Title: "Go 1.24 发布", Source: "Go Blog", Summary: "详细变更说明"},
		{Title: "NewsPocket 升级", Source: "GitHub", Summary: "全面重构"},
	}

	digest, err := client.GenerateDailyDigest(context.Background(), items)
	if err != nil {
		t.Fatalf("GenerateDailyDigest failed: %v", err)
	}

	if digest != expectedResponse {
		t.Fatalf("expected %q, got %q", expectedResponse, digest)
	}
}

func TestGenerateDailyDigestServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"Internal Server Error"}}`))
	}))
	defer server.Close()

	client := NewClientWithConfig(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "mock-model",
		Timeout: 5 * time.Second,
	})

	_, err := client.GenerateDailyDigest(context.Background(), []parser.NewsItem{
		{Title: "Test", Source: "RSS"},
	})
	if err == nil {
		t.Fatal("expected error on 500 status code, got nil")
	}
}
