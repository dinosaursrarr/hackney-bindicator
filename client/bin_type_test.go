package client_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/iotest"
	"time"

	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/assert"
)

func TestBadBinTypeUrl(t *testing.T) {
	badUrl, _ := url.Parse("ftp://foo.bar")
	client := client.BinsClient{http.Client{}, nil, badUrl, nil}

	res, err := client.GetBinType(BinId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestSetUserAgentGettingBinType(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["User-Agent"])
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	client.GetBinType(BinId)
}

func TestSetAcceptGettingBinType(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["Accept"])
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	client.GetBinType(BinId)
}

func TestHttpErrorGettingBinType(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	httpClient := http.Client{
		Transport: fakeRoundTripper{
			Fn: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("foo")
			},
		},
	}
	client := client.BinsClient{httpClient, nil, apiUrl, nil}

	res, err := client.GetBinType(BinId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "foo")
}

func TestBadStatusCodeGettingBinType(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := client.GetBinType(BinId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Status code 418")
}

func TestErrorReadingBinType(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	httpClient := http.Client{
		Transport: fakeRoundTripper{
			Fn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(iotest.ErrReader(errors.New("nope"))),
				}, nil
			},
		},
	}
	client := client.BinsClient{httpClient, nil, apiUrl, nil}

	res, err := client.GetBinType(BinId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "nope")
}

func TestBinTypeNotFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := client.GetBinType(BinId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Bin type not found")
}

func TestEmptyBinTypeFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"subTitle": ""
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := client.GetBinType(BinId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Bin type not found")
}

func TestSuccessBinType(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"subTitle": "Garbage sack",
				"binType": "5f96b455e36673006420c529"
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := binsClient.GetBinType(BinId)

	assert.Equal(t, client.BinType{Name: "Garbage sack", Type: client.Food}, res)
	assert.Nil(t, err)
}

func TestSuccessBinTypeName(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"subTitle": "Garbage sack"
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := binsClient.GetBinType(BinId)

	assert.Equal(t, res, client.BinType{Name: "Garbage sack"})
	assert.Nil(t, err)
}

func TestSuccessBinTypeType(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"binType": "5f96b455e36673006420c529"
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := binsClient.GetBinType(BinId)

	assert.Equal(t, res, client.BinType{Type: client.Food})
	assert.Nil(t, err)
}

func TestFetchBinTypeTwiceWithoutCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"subTitle": "Garbage sack",
				"binType": "5f96b455e36673006420c529"
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	client.GetBinType(BinId)
	client.GetBinType(BinId)

	assert.Equal(t, fetches, 2)
}

func TestFetchBinTypeOnceWithCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"subTitle": "Garbage sack",
				"binType": "5f96b455e36673006420c529"
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	client := client.BinsClient{http.Client{}, nil, apiUrl, cache}

	client.GetBinType(BinId)
	client.GetBinType(BinId)

	assert.Equal(t, fetches, 1)
}
