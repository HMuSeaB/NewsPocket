package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/ai"
	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/digest"
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

// OPML 格式解析结构体
type opmlDoc struct {
	XMLName xml.Name    `xml:"opml"`
	Version string      `xml:"version,attr"`
	Head    opmlHead    `xml:"head"`
	Body    opmlBody    `xml:"body"`
}

type opmlHead struct {
	Title string `xml:"title"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	Type     string        `xml:"type,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	HTMLURL  string        `xml:"htmlUrl,attr"`
	Category string        `xml:"category,attr"`
	Outlines []opmlOutline `xml:"outline"`
}

// SelectAndImportOPML 弹出文件选择器并导入 OPML 订阅源
func (a *App) SelectAndImportOPML() (int, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 OPML 订阅文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "OPML / XML Files (*.opml;*.xml)",
				Pattern:     "*.opml;*.xml",
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("打开文件对话框失败: %w", err)
	}
	if path == "" {
		return 0, nil // 用户取消
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("读取 OPML 文件失败: %w", err)
	}

	return a.ImportOPML(string(content))
}

// ImportOPML 解析 OPML XML 字符串并去重合并到当前 sources.json
func (a *App) ImportOPML(xmlContent string) (int, error) {
	var doc opmlDoc
	if err := xml.Unmarshal([]byte(xmlContent), &doc); err != nil {
		return 0, fmt.Errorf("解析 OPML XML 失败: %w", err)
	}

	// 提取所有 RSS 节点
	var extracted []config.Source
	var extractOutlines func(nodes []opmlOutline, parentCat string)
	extractOutlines = func(nodes []opmlOutline, parentCat string) {
		for _, node := range nodes {
			cat := parentCat
			if node.Category != "" {
				cat = node.Category
			} else if len(node.Outlines) > 0 && node.Text != "" {
				cat = node.Text
			}

			if node.XMLURL != "" {
				name := node.Title
				if name == "" {
					name = node.Text
				}
				if name == "" {
					if u, err := url.Parse(node.XMLURL); err == nil {
						name = u.Host
					} else {
						name = "未命名 RSS"
					}
				}

				if cat == "" {
					cat = "行业动态"
				}

				enabled := true
				extracted = append(extracted, config.Source{
					Name:      name,
					URL:       node.XMLURL,
					Category:  cat,
					Type:      "rss",
					Enabled:   &enabled,
					Collapsed: false,
				})
			}

			if len(node.Outlines) > 0 {
				extractOutlines(node.Outlines, cat)
			}
		}
	}

	extractOutlines(doc.Body.Outlines, "")

	if len(extracted) == 0 {
		return 0, fmt.Errorf("OPML 文件中未找到有效的 RSS 订阅链接")
	}

	// 读取当前配置并去重合并
	cfgStr, err := a.GetConfig()
	if err != nil {
		return 0, err
	}

	var currentCfg config.Config
	_ = json.Unmarshal([]byte(cfgStr), &currentCfg)

	existingURLs := make(map[string]bool)
	for _, src := range currentCfg.Sources {
		existingURLs[strings.TrimSpace(src.URL)] = true
	}

	addedCount := 0
	for _, newSrc := range extracted {
		cleanURL := strings.TrimSpace(newSrc.URL)
		if cleanURL != "" && !existingURLs[cleanURL] {
			existingURLs[cleanURL] = true
			currentCfg.Sources = append(currentCfg.Sources, newSrc)
			addedCount++
		}
	}

	if addedCount > 0 {
		newJSON, err := json.MarshalIndent(currentCfg, "", "  ")
		if err != nil {
			return 0, fmt.Errorf("序列化合并配置失败: %w", err)
		}
		if err := a.SaveConfig(string(newJSON)); err != nil {
			return 0, fmt.Errorf("保存新配置失败: %w", err)
		}
	}

	slog.Info("OPML 导入成功", "added", addedCount, "total_extracted", len(extracted))
	return addedCount, nil
}

// ExportOPML 将当前启用的 RSS 源导出为标准 OPML 2.0 XML
func (a *App) ExportOPML() (string, error) {
	cfgStr, err := a.GetConfig()
	if err != nil {
		return "", err
	}

	var cfg config.Config
	if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
		return "", fmt.Errorf("解析配置失败: %w", err)
	}

	doc := opmlDoc{
		Version: "2.0",
		Head: opmlHead{
			Title: "NewsPocket Feeds Export",
		},
	}

	// 按分类组织 outlines
	catMap := make(map[string][]opmlOutline)
	for _, src := range cfg.Sources {
		if src.Type == "rss" || src.Type == "" {
			cat := src.Category
			if cat == "" {
				cat = "默认分类"
			}
			catMap[cat] = append(catMap[cat], opmlOutline{
				Text:     src.Name,
				Title:    src.Name,
				Type:     "rss",
				XMLURL:   src.URL,
				Category: cat,
			})
		}
	}

	for cat, outlines := range catMap {
		doc.Body.Outlines = append(doc.Body.Outlines, opmlOutline{
			Text:     cat,
			Title:    cat,
			Outlines: outlines,
		})
	}

	xmlBytes, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("生成 OPML XML 失败: %w", err)
	}

	return xml.Header + string(xmlBytes), nil
}

// PreviewEmail 在内存中抓取并渲染完整 HTML 邮件内容以供前端 iframe 实时预览
func (a *App) PreviewEmail() (string, error) {
	slog.Info("开始生成邮件实时预览")

	cfgStr, err := a.GetConfig()
	if err != nil {
		return "", fmt.Errorf("读取配置失败: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
		return "", fmt.Errorf("解析配置失败: %w", err)
	}

	result, err := digest.Build(&cfg, ai.NewClientFromSettings(cfg.Settings.AI), digest.Options{
		FetchTimeout: 15 * time.Second,
		TitleSuffix:  " (实时预览)",
	})
	if err != nil {
		switch {
		case errors.Is(err, digest.ErrNoSources):
			return previewNotice("暂无可用的已启用新闻源", "请至少启用一个新闻源后再预览。"), nil
		case errors.Is(err, digest.ErrNoResults):
			return previewNotice("未抓取到任何内容", "请检查网络连接或源的配置有效性。"), nil
		case errors.Is(err, digest.ErrNoItems):
			return previewNotice("解析后无有效内容", "可能所有抓取到的内容都在设定时间窗口（如 24 小时）之外。"), nil
		default:
			return "", fmt.Errorf("晨报生成失败: %w", err)
		}
	}

	return result.HTML, nil
}

// previewNotice 生成预览弹窗内的居中提示卡片 HTML（无内容/异常时的降级展示）
func previewNotice(title, detail string) string {
	return fmt.Sprintf(
		`<div style="padding:30px;text-align:center;color:#666;font-family:sans-serif;">`+
			`<h3>⚠️ %s</h3><p>%s</p></div>`, title, detail)
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

	var previewBuilder strings.Builder
	if len(items) > 0 {
		limit := 5
		if len(items) < limit {
			limit = len(items)
		}
		previewBuilder.WriteString(fmt.Sprintf("✅ 成功抓取！共获得 **%d** 条有效新闻。\n\n--- 预览前 %d 条 ---\n\n", len(items), limit))
		for i := 0; i < limit; i++ {
			item := items[i]
			previewBuilder.WriteString(fmt.Sprintf("%d. [%s]\n   %s (%s)\n\n", i+1, item.Title, item.Link, item.Time))
		}
	} else {
		// 检查抓取到了多少条原始数据，用以排查过滤原因
		rawCount := len(result.Entries)
		if rawCount > 0 {
			previewBuilder.WriteString(fmt.Sprintf("⚠️ 抓取成功，解析到 **%d** 条原始数据，但由于 24 小时时间过滤器或清洗规则，最终有效新闻为 **0** 条。\n\n", rawCount))
			previewBuilder.WriteString("💡 **可能的原因**：\n")
			previewBuilder.WriteString("1. 该源最近 24 小时内没有任何更新。\n")
			previewBuilder.WriteString("2. `time_field` 字段解析出来的时间不匹配，或者格式未被识别，导致其被判断为过期。\n\n")
			previewBuilder.WriteString("--- 📋 原始数据首条分析 ---\n\n")

			firstEntry := result.Entries[0]
			previewBuilder.WriteString(fmt.Sprintf("* **标题 (Title)**: %s\n", firstEntry.Title))
			previewBuilder.WriteString(fmt.Sprintf("* **链接 (Link)**: %s\n", firstEntry.Link))

			// 尝试输出原始的时间字段值
			var rawTimeVal any
			if src.JSONConfig != nil && src.JSONConfig.TimeField != "" {
				keys := strings.Split(src.JSONConfig.TimeField, ".")
				var curr any = firstEntry.RawData
				for _, k := range keys {
					if m, ok := curr.(map[string]any); ok {
						curr = m[k]
					} else {
						curr = nil
						break
					}
				}
				rawTimeVal = curr
			}

			previewBuilder.WriteString(fmt.Sprintf("* **时间字段 (TimeField 配置)**: `%s`\n", src.JSONConfig.TimeField))
			previewBuilder.WriteString(fmt.Sprintf("* **提取到的原始时间值**: `%v`\n", rawTimeVal))
			previewBuilder.WriteString(fmt.Sprintf("* **解析后的实际时间 (Go)**: `%s` (若为 0001-01-01 表示解析失败)\n", firstEntry.Published.Format("2006-01-02 15:04:05 MST")))
		} else {
			previewBuilder.WriteString("❌ 抓取完成，但解析到的条目数量为 **0**。\n\n")
			previewBuilder.WriteString("💡 **建议排查**：\n")
			if src.Type == "json_api" && src.JSONConfig != nil {
				previewBuilder.WriteString(fmt.Sprintf("1. 确认接口返回的 JSON 中，在 `items_path` (`%s`) 对应位置确实存在有效数组列表。\n", src.JSONConfig.ItemsPath))
				previewBuilder.WriteString(fmt.Sprintf("2. 确认 `title_field` (`%s`) 拼写及路径是否正确。\n", src.JSONConfig.TitleField))
			} else {
				previewBuilder.WriteString("1. 检查 RSS 源地址在浏览器中是否能正常访问并返回合法的 XML/RSS 数据。\n")
			}
		}
	}

	return previewBuilder.String(), nil
}
