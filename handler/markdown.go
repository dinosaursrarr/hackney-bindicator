package handler

import (
	"net/http"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type MarkdownHandler struct {
	Markdown []byte
	Title    string
	CssPath  string
}

func (h *MarkdownHandler) Handle(w http.ResponseWriter, r *http.Request) {
	htmlFlags := html.CompletePage | html.UseXHTML | html.CommonFlags
	opts := html.RendererOptions{
		Flags: htmlFlags,
		Title: h.Title,
		CSS:   h.CssPath,
	}
	renderer := html.NewRenderer(opts)

	html := markdown.ToHTML(h.Markdown, nil, renderer)
	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
}
