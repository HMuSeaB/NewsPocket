package fetcher

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
