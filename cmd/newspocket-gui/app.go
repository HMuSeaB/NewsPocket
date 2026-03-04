package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/fetcher"
	"github.com/HMuSeaB/NewsPocket/internal/parser"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	configPath string
}

// NewApp creates a new App application struct
func NewApp() *App {
	// 尝试寻找根目录下的 config/sources.json
	exePath, err := os.Executable()
	var configPath string
	if err == nil {
		exeDir := filepath.Dir(exePath)
		cwd, _ := os.Getwd()

		// 穷举可能的配置路径 (考虑开发环境 / 生产打包的不同目录深度)
		pathsToTry := []string{
			filepath.Join(exeDir, "config", "sources.json"),       // 生产环境同级
			filepath.Join(exeDir, "..", "config", "sources.json"), // 生产环境上一级(如在 bin/ 下)
			filepath.Join(exeDir, "..", "..", "config", "sources.json"),
			filepath.Join(cwd, "config", "sources.json"),             // 开发环境当前目录下的 config
			filepath.Join(cwd, "..", "config", "sources.json"),       // 开发环境上一级
			filepath.Join(cwd, "..", "..", "config", "sources.json"), // wails dev 的目录结构
		}

		for _, p := range pathsToTry {
			if _, err := os.Stat(p); err == nil {
				configPath = p
				break
			}
		}
	}

	if configPath == "" {
		// 最终的 fallback
		configPath = "config/sources.json"
	}

	return &App{
		configPath: configPath,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetConfig returns the contents of the sources.json
func (a *App) GetConfig() (string, error) {
	slog.Info("读取配置文件", "path", a.configPath)
	content, err := os.ReadFile(a.configPath)
	if err != nil {
		// 如果文件不存在，返回空的模板
		if os.IsNotExist(err) {
			slog.Warn("配置文件不存在，返回空配置")
			return `{"sources":[]}`, nil
		}
		return "", fmt.Errorf("读取配置文件失败: %w", err)
	}
	return string(content), nil
}

// SelectConfigFile prompts the user to select the source.json file and returns its content
func (a *App) SelectConfigFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 config/sources.json",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON Files (*.json)",
				Pattern:     "*.json",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if path == "" {
		return "", nil // 用户取消
	}

	a.configPath = path
	return a.GetConfig()
}

// SaveConfig saves the JSON string to sources.json
func (a *App) SaveConfig(jsonStr string) error {
	slog.Info("保存配置文件", "path", a.configPath)

	// 格式化 JSON
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return fmt.Errorf("无效的 JSON 格式: %w", err)
	}

	formattedJson, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("格式化 JSON 失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(a.configPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(a.configPath, formattedJson, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// TestSource tests a single source and returns a preview of the fetched items
func (a *App) TestSource(sourceJson string) (string, error) {
	var src config.Source
	if err := json.Unmarshal([]byte(sourceJson), &src); err != nil {
		return "", fmt.Errorf("解析 Source JSON 失败: %w", err)
	}

	slog.Info("测试抓取源", "name", src.Name, "type", src.Type)

	f := fetcher.New(15 * time.Second)
	result, err := f.FetchSingle(src)

	if err != nil {
		return "", fmt.Errorf("抓取失败: %w", err)
	}

	p := parser.New(300, 24)
	items := p.ParseAll([]fetcher.FetchResult{*result}, 15)

	// 构造返回值，只取前 5 条作为预览
	var previewBuilder strings.Builder
	previewBuilder.WriteString(fmt.Sprintf("✅ 成功抓取！共获得 **%d** 条有效新闻。\n\n--- 预览前 5 条 ---\n\n", len(items)))

	limit := 5
	if len(items) < limit {
		limit = len(items)
	}

	for i := 0; i < limit; i++ {
		item := items[i]
		previewBuilder.WriteString(fmt.Sprintf("%d. [%s]\n   %s (%s)\n\n", i+1, item.Title, item.Link, item.Time))
	}

	return previewBuilder.String(), nil
}
