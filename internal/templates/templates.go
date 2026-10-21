package templates

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
)

func RenderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(md))
	}
	return template.HTML(buf.String())
}

// html/template limitation: custom functions must be added to FuncMap before parsing template strings
func InitTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"renderMD": RenderMarkdown,
	}

	tmpl, err := template.New("root").Funcs(funcMap).Parse(HTMLTemplates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return tmpl, nil
}

func Parse() (*template.Template, error) {
	return InitTemplates()
}

