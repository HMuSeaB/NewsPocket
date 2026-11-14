// Package history 提供轻量级跨日持久化去重功能。
// 记录最近 N 天内已推送过的文章哈希/链接，防止长热点连日重复出现在每日早报中。
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

// Store 持久化去重库
type Store struct {
	mu        sync.RWMutex
	filePath  string
	retention time.Duration

	Version     int              `json:"version"`
	LastUpdated time.Time        `json:"last_updated"`
	Records     map[string]int64 `json:"records"` // key: hash, value: unix timestamp
}

// OpenStore 打开或初始化一个去重存储库，并自动清理过期记录
func OpenStore(path string, retentionDays int) (*Store, error) {
	if retentionDays <= 0 {
		retentionDays = 7 // 默认记忆 7 天
	}
	retention := time.Duration(retentionDays) * 24 * time.Hour

	s := &Store{
		filePath:  path,
		retention: retention,
		Version:   1,
		Records:   make(map[string]int64),
	}

	data, err := os.ReadFile(path)
	if err == nil {
		if jsonErr := json.Unmarshal(data, s); jsonErr != nil {
			slog.Warn("历史去重文件解析损坏，将重新初始化", "path", path, "error", jsonErr)
			s.Records = make(map[string]int64)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取历史去重文件失败: %w", err)
	}

	// 自动清理过期记录
	s.purgeExpired()

	return s, nil
}

// computeHash 计算条目的稳定哈希
func computeHash(link, title string) string {
	target := link
	if target == "" {
		target = title
	}
	h := sha256.Sum256([]byte(target))
	return hex.EncodeToString(h[:16]) // 取前 32 位 hex
}

// IsSeen 判断该条目在保留期内是否已被推送过
func (s *Store) IsSeen(item parser.NewsItem) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash := computeHash(item.Link, item.Title)
	ts, exists := s.Records[hash]
	if !exists {
		return false
	}

	// 检查是否在保留期内
	if time.Since(time.Unix(ts, 0)) > s.retention {
		return false
	}
	return true
}

// FilterUnseen 从条目列表中过滤掉已推送过的内容
func (s *Store) FilterUnseen(items []parser.NewsItem) (fresh []parser.NewsItem, skipped int) {
	for _, item := range items {
		if s.IsSeen(item) {
			skipped++
		} else {
			fresh = append(fresh, item)
		}
	}
	return fresh, skipped
}

// RecordItems 记录本次推送的条目
func (s *Store) RecordItems(items []parser.NewsItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	for _, item := range items {
		hash := computeHash(item.Link, item.Title)
		s.Records[hash] = now
	}
	s.LastUpdated = time.Now()
}

// purgeExpired 清理超过保留周期的老旧记录
func (s *Store) purgeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.retention).Unix()
	removed := 0
	for h, ts := range s.Records {
		if ts < cutoff {
			delete(s.Records, h)
			removed++
		}
	}
	if removed > 0 {
		slog.Debug("已清理过期历史推送记录", "count", removed)
	}
}

// Save 持久化保存到文件
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filePath == "" {
		return nil
	}

	dir := filepath.Dir(s.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建历史目录失败: %w", err)
		}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化历史记录失败: %w", err)
	}

	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时历史文件失败: %w", err)
	}

	if err := os.Rename(tmpFile, s.filePath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("原子替换历史文件失败: %w", err)
	}

	return nil
}
