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
