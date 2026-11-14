package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

func TestHistoryStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "history_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "history.json")

	// 1. 初始化
	store, err := OpenStore(dbPath, 7)
	if err != nil {
		t.Fatalf("OpenStore 失败: %v", err)
	}

	items := []parser.NewsItem{
		{Title: "Go 1.25 发布", Link: "https://go.dev/doc/go1.25"},
		{Title: "NewsPocket 升级", Link: "https://github.com/HMuSeaB/NewsPocket"},
	}

	// 初始状态都不应存在
	for _, it := range items {
		if store.IsSeen(it) {
			t.Errorf("初始条目不应已被记录: %s", it.Title)
		}
	}

	// 2. 记录条目并存盘
	store.RecordItems(items)
	for _, it := range items {
		if !store.IsSeen(it) {
			t.Errorf("记录后应为已见状态: %s", it.Title)
		}
	}

	if err := store.Save(); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	// 3. 重新加载验证持久化
	reloaded, err := OpenStore(dbPath, 7)
	if err != nil {
		t.Fatalf("重新加载 OpenStore 失败: %v", err)
	}

	testItems := []parser.NewsItem{
		{Title: "Go 1.25 发布", Link: "https://go.dev/doc/go1.25"},       // 重复
		{Title: "全新开源项目", Link: "https://github.com/example/new"}, // 全新
	}

	fresh, skipped := reloaded.FilterUnseen(testItems)
	if skipped != 1 {
		t.Errorf("应该过滤掉 1 个重复条目，实际过滤: %d", skipped)
	}
	if len(fresh) != 1 || fresh[0].Title != "全新开源项目" {
		t.Errorf("过滤后保留的新鲜条目不符合预期: %+v", fresh)
	}

	// 4. 过期清理验证
	reloaded.Records["old_hash"] = time.Now().Add(-10 * 24 * time.Hour).Unix()
	reloaded.purgeExpired()
	if _, exists := reloaded.Records["old_hash"]; exists {
		t.Errorf("超过保留周期的老旧记录应被清理")
	}
}
