package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportAndExportOPML(t *testing.T) {
	// 创建临时 sources.json
	tempDir := t.TempDir()
	tempConfigPath := filepath.Join(tempDir, "sources.json")
	if err := os.WriteFile(tempConfigPath, []byte(`{"sources":[]}`), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	app := &App{
		configPath: tempConfigPath,
	}

	sampleOPML := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test Feeds</title></head>
  <body>
    <outline text="科技博客" title="科技博客">
      <outline type="rss" text="Simon Willison" title="Simon Willison" xmlUrl="https://simonwillison.net/atom/everything/" htmlUrl="https://simonwillison.net"/>
      <outline type="rss" text="Jeff Geerling" title="Jeff Geerling" xmlUrl="https://www.jeffgeerling.com/blog.xml" htmlUrl="https://jeffgeerling.com"/>
    </outline>
  </body>
</opml>`

	importedCount, err := app.ImportOPML(sampleOPML)
	if err != nil {
		t.Fatalf("ImportOPML failed: %v", err)
	}
	if importedCount != 2 {
		t.Fatalf("expected 2 imported feeds, got %d", importedCount)
	}

	// 再次导入相同的 OPML，验证去重能力
	reimportedCount, err := app.ImportOPML(sampleOPML)
	if err != nil {
		t.Fatalf("Second ImportOPML failed: %v", err)
	}
	if reimportedCount != 0 {
		t.Fatalf("expected 0 re-imported feeds due to deduplication, got %d", reimportedCount)
	}

	// 验证导出 OPML
	exportedOPML, err := app.ExportOPML()
	if err != nil {
		t.Fatalf("ExportOPML failed: %v", err)
	}

	if !strings.Contains(exportedOPML, "https://simonwillison.net/atom/everything/") {
		t.Errorf("exported OPML missing feed URL")
	}
	if !strings.Contains(exportedOPML, "科技博客") {
		t.Errorf("exported OPML missing category outline")
	}
}
