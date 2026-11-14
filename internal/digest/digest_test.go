package digest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/history"
)

func TestDigestBuild(t *testing.T) {
	// Mock RSS 服务
	mockRSS := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<title>Test Feed</title>
	<item>
		<title>News 1</title>
		<link>https://example.com/news1?utm_source=rss</link>
		<description>Summary 1</description>
		<pubDate>` + time.Now().Format(time.RFC1123) + `</pubDate>
	</item>
</channel>
</rss>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(mockRSS))
	}))
	defer ts.Close()

	cfg := &config.Config{
		Sources: []config.Source{
			{
				Name:     "Test Source",
				URL:      ts.URL,
				Category: "科技",
				Type:     "rss",
			},
		},
		Settings: config.Settings{
			MaxItemsPerSource: 5,
			HoursLookback:     24,
			SummaryMaxLength:  200,
		},
	}

	tempDir, err := os.MkdirTemp("", "digest_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	histStore, err := history.OpenStore(filepath.Join(tempDir, "hist.json"), 7)
	if err != nil {
		t.Fatalf("OpenStore 失败: %v", err)
	}

	res, err := Build(cfg, nil, Options{
		FetchTimeout: 5 * time.Second,
		TitleSuffix:  " (测试)",
		HistoryStore: histStore,
	})
	if err != nil {
		t.Fatalf("Build 失败: %v", err)
	}

	if res.TotalCount != 1 {
		t.Errorf("TotalCount 应为 1，实际为 %d", res.TotalCount)
	}
	if res.HTML == "" {
		t.Errorf("HTML 输出不能为空")
	}
	if res.Markdown == "" {
		t.Errorf("Markdown 输出不能为空")
	}
	if len(res.SourceReports) != 1 {
		t.Errorf("SourceReports 长度应为 1")
	}
	if res.SourceReports[0].Duration <= 0 {
		t.Errorf("抓取耗时应大于 0")
	}
}
