package handler_test

import (
	"github.com/dinosaursrarr/hackney-bindicator/handler"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownEmpty(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	handler := handler.MarkdownHandler{}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t, w.Body.String(), "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">\n<html xmlns=\"http://www.w3.org/1999/xhtml\">\n<head>\n  <title></title>\n  <meta name=\"GENERATOR\" content=\"github.com/gomarkdown/markdown markdown processor for Go\" />\n  <meta charset=\"utf-8\" />\n</head>\n<body>\n\n\n</body>\n</html>\n")
}

func TestMarkdownExists(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	md := []byte("# Title\n\nparagraph\n")
	handler := handler.MarkdownHandler{Markdown: md}

	handler.Handle(w, r)

	assert.Contains(t, w.Body.String(), "<h1>Title</h1>")
	assert.Contains(t, w.Body.String(), "<p>paragraph</p>")
}

func TestSetTitle(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	md := []byte("# Title\n\nparagraph\n")
	handler := handler.MarkdownHandler{Markdown: md, Title: "foo"}

	handler.Handle(w, r)

	assert.Contains(t, w.Body.String(), "<title>foo</title>")
}

func TestCssPathExists(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	md := []byte("# Title\n\nparagraph\n")
	handler := handler.MarkdownHandler{Markdown: md, CssPath: "foo"}

	handler.Handle(w, r)

	assert.Contains(t, w.Body.String(), "<link rel=\"stylesheet\" type=\"text/css\" href=\"foo\" />")
}

func TestMarkdownSetContentTypeHeader(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	handler := handler.MarkdownHandler{}

	handler.Handle(w, r)

	assert.Contains(t, w.Header(), "Content-Type")
}
