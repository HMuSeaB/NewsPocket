// Package renderer 负责使用 Go 模板引擎渲染 HTML 邮件。
package renderer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

//go:embed templates/email.gohtml
var templateFS embed.FS

// TemplateData 邮件模板的上下文数据
type TemplateData struct {
	Title         string
	Date          string
	TotalCount    int
	SourceCount   int
	CategoryCount int
	Sections      []parser.CategorySection
}

// Renderer HTML 模板渲染器
type Renderer struct {
}

// New 创建渲染器
func New() *Renderer {
	return &Renderer{}
}

// Render 渲染邮件模板
func (r *Renderer) Render(data TemplateData) (string, error) {
	// 自定义模板函数
	funcMap := template.FuncMap{
		"categoryIcon": categoryIcon,
		"sub":          func(a, b int) int { return a - b },
	}

	tmpl, err := template.New("email.gohtml").Funcs(funcMap).ParseFS(templateFS, "templates/email.gohtml")
	if err != nil {
		return "", fmt.Errorf("模板解析失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("模板渲染失败: %w", err)
	}

	return buf.String(), nil
}

// categoryIcon 返回分类对应的 SVG 图标 HTML
func categoryIcon(category string) template.HTML {
	svgStyle := `width="18" height="18" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" style="width: 18px; height: 18px; display: block; margin: 0 auto;"`

	switch category {
	case "行业动态":
		return template.HTML(fmt.Sprintf(
			`<td class="section-icon" width="36" height="36"><svg %s fill="#7c3aed"><path d="M3 21h18v-2H3v2zm0-4h18v-2H3v2zm0-4h18v-2H3v2zm0-4h18V7H3v2zm0-6v2h18V3H3z"/></svg></td>`,
			svgStyle))
	case "全球热点":
		return template.HTML(fmt.Sprintf(
			`<td class="section-icon" width="36" height="36"><svg %s fill="#7c3aed"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg></td>`,
			svgStyle))
	case "科技生活":
		return template.HTML(fmt.Sprintf(
			`<td class="section-icon section-icon-orange" width="36" height="36"><svg %s fill="#ea580c"><path d="M7 2v11h3v9l7-12h-4l4-8z"/></svg></td>`,
			svgStyle))
	case "社交热点":
		return template.HTML(fmt.Sprintf(
			`<td class="section-icon" width="36" height="36"><svg %s fill="#7c3aed"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H6l-2 2V4h16v12z"/></svg></td>`,
			svgStyle))
	default:
		return template.HTML(fmt.Sprintf(
			`<td class="section-icon" width="36" height="36"><svg %s fill="#7c3aed"><path d="M12 2l-5.5 9h11L12 2zm0 3.84L13.93 9h-3.87L12 5.84zM17.5 13c-2.49 0-4.5 2.01-4.5 4.5s2.01 4.5 4.5 4.5 4.5-2.01 4.5-4.5-2.01-4.5-4.5-4.5zm0 7c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5zM3 21.5h8v-8H3v8zm2-6h4v4H5v-4z"/></svg></td>`,
			svgStyle))
	}
}
