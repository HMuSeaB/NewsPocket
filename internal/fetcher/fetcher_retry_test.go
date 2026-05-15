package fetcher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
)

func withNoRetryDelay(t *testing.T) {
	t.Helper()

	original := retryDelay
	retryDelay = func(int) time.Duration {
		return 0
	}
	t.Cleanup(func() {
		retryDelay = original
	})
}

func TestDoRequestWithRetryRetriesServerErrors(t *testing.T) {
	withNoRetryDelay(t)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests < 3 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	resp, err := doRequestWithRetry(context.Background(), server.Client(), config.Source{
		Name: "retry-5xx",
		URL:  server.URL,
	}, http.MethodGet)
	if err != nil {
		t.Fatalf("doRequestWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()

	if requests != 3 {
		t.Fatalf("expected 3 requests, got %d", requests)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", resp.StatusCode)
	}
}

func TestDoRequestWithRetryDoesNotRetryClientErrors(t *testing.T) {
	withNoRetryDelay(t)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	resp, err := doRequestWithRetry(context.Background(), server.Client(), config.Source{
		Name: "no-retry-4xx",
		URL:  server.URL,
	}, http.MethodGet)
	if err != nil {
		t.Fatalf("doRequestWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()

	if requests != 1 {
		t.Fatalf("expected 1 request, got %d", requests)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 response, got %d", resp.StatusCode)
	}
}

func TestDoRequestWithRetryReplaysPostBody(t *testing.T) {
	withNoRetryDelay(t)

	wantBody := `{"hello":"world"}`
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		bodies = append(bodies, string(body))

		if len(bodies) == 1 {
			http.Error(w, "temporary", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	body, err := json.Marshal(map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	resp, err := doRequestWithRetry(context.Background(), server.Client(), config.Source{
		Name: "post-replay",
		URL:  server.URL,
		Body: body,
	}, http.MethodPost)
	if err != nil {
		t.Fatalf("doRequestWithRetry returned error: %v", err)
	}
	defer resp.Body.Close()

	if len(bodies) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(bodies))
	}
	for i, got := range bodies {
		if got != wantBody {
			t.Fatalf("request %d body = %q, want %q", i+1, got, wantBody)
		}
	}
}
