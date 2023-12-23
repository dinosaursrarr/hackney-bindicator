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
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestBadWorkflowIdUrl(t *testing.T) {
	badUrl, _ := url.Parse("ftp://foo.bar")
	client := client.BinsClient{http.Client{}, nil, badUrl, nil, nil}

	res, err := client.GetBinWorkflowId(BinId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestSetAccessTokenGettingWorkflowId(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header["Authorization"], "Bearer "+Token)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	client.GetBinWorkflowId(BinId, Token)
}

func TestHttpErrorGettingWorkflowId(t *testing.T) {
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
	client := client.BinsClient{httpClient, nil, apiUrl, nil, nil}

	res, err := client.GetBinWorkflowId(BinId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "foo")
}

func TestBadStatusCodeGettingWorkflowId(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinWorkflowId(BinId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Status code 418")
}

func TestErrorReadingWorkflowId(t *testing.T) {
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
	client := client.BinsClient{httpClient, nil, apiUrl, nil, nil}

	res, err := client.GetBinWorkflowId(BinId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "nope")
}

func TestWorkflowIdNotFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinWorkflowId(BinId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Workflow ID not found")
}

func TestEmptyWorkflowIdFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"attributes": [
							{
								"attributeCode": "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec",
								"value": ""
							}
						]
					}
				]
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinWorkflowId(BinId, Token)

	assert.Empty(t, res)
	assert.Nil(t, err)
}

func TestSuccessWorkflowId(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"attributes": [
							{
								"attributeCode": "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec",
								"value": "foo"
							}
						]
					}
				]
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinWorkflowId(BinId, Token)

	assert.Equal(t, res, "foo")
	assert.Nil(t, err)
}

func TestFetchWorkflowIdTwiceWithoutCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"attributes": [
							{
								"attributeCode": "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec",
								"value": "foo"
							}
						]
					}
				]
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	client.GetBinWorkflowId(BinId, Token)
	client.GetBinWorkflowId(BinId, Token)

	assert.Equal(t, fetches, 2)
}

func TestFetchWorkflowIdOnceWithCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"attributes": [
							{
								"attributeCode": "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec",
								"value": "foo"
							}
						]
					}
				]
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	cache := cache.New(15*time.Minute, 30*time.Minute)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, cache}

	client.GetBinWorkflowId(BinId, Token)
	client.GetBinWorkflowId(BinId, Token)

	assert.Equal(t, fetches, 1)
}
