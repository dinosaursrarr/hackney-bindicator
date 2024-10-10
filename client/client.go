package client

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jonboulle/clockwork"
)

const addressUrl = "/property/opensearch"
const binIdUrl = "/alloywastepages/getproperty/"
const binTypeUrl = "/alloywastepages/getbin/"
const workflowIdUrl = "/alloywastepages/getcollection/"
const scheduleUrl = "/alloywastepages/getworkflow/"
const userAgent = "github.com/dinosaursrarr/hackney-bindicator"

func tidy(s string) string {
	return space.ReplaceAllString(strings.TrimSpace(s), " ")
}

type BinsClient struct {
	HttpClient http.Client
	Clock      clockwork.Clock
	ApiHost    *url.URL
	Cache      *expirable.LRU[string, interface{}]
}
