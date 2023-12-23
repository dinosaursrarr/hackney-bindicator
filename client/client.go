package client

import (
	"net/http"
	"net/url"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jonboulle/clockwork"
)

const itemUrl = "/api/item/"
const queryUrl = "/api/aqs/query"
const workflowUrl = "/api/workflow/"

type BinsClient struct {
	HttpClient http.Client
	Clock      clockwork.Clock
	ApiHost    *url.URL
	StartUrl   *url.URL
	Cache      *expirable.LRU[string, interface{}]
}
