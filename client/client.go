package client

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jonboulle/clockwork"
)

const itemUrl = "/api/item/"
const queryUrl = "/api/aqs/query"
const workflowUrl = "/api/workflow/"
const userAgent = "github.com/dinosaursrarr/hackney-bindicator"

func tidy(s string) string {
	return space.ReplaceAllString(strings.TrimSpace(s), " ")
}

type BinsClient struct {
	HttpClient http.Client
	Clock      clockwork.Clock
	ApiHost    *url.URL
	StartUrl   *url.URL
	Cache      *expirable.LRU[string, interface{}]
}
