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

func TestBadBinIdUrl(t *testing.T) {
	badUrl, _ := url.Parse("ftp://foo.bar")
	client := client.BinsClient{http.Client{}, nil, badUrl, nil, nil}

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestSetAccessTokenGettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header["Authorization"], "Bearer "+Token)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	client.GetBinIds(PropertyId, Token)
}

func TestSetUserAgentGettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["User-Agent"])
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	client.GetBinIds(PropertyId, Token)
}

func TestHttpErrorGettingBinIds(t *testing.T) {
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

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "foo")
}

func TestStatusCode400GettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Status code 400")
}

func TestBadStatusCodeGettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Status code 418")
}

func TestErrorReadingBinIds(t *testing.T) {
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

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "nope")
}

func TestBinIdsNotFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Bin IDs not found")
}

func TestNoBinIdsFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"item": {
					"attributes": [
						{
							"attributeCode": "attributes_wasteContainersAssignableWasteContainers",
							"value": []
						}
					]
				}
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Empty(t, res)
	assert.Nil(t, err)
}

func TestSuccessBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"item": {
					"attributes": [
						{
							"attributeCode": "attributes_wasteContainersAssignableWasteContainers",
							"value": [
								"foo",
								"bar",
								"baz"
							]
						}
					]
				}
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetBinIds(PropertyId, Token)

	assert.Equal(t, res, []string{"foo", "bar", "baz"})
	assert.Nil(t, err)
}

func TestFetchBinIdsTwiceWithoutCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"item": {
					"attributes": [
						{
							"attributeCode": "attributes_wasteContainersAssignableWasteContainers",
							"value": [
								"foo",
								"bar",
								"baz"
							]
						}
					]
				}
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	client.GetBinIds(PropertyId, Token)
	client.GetBinIds(PropertyId, Token)

	assert.Equal(t, fetches, 2)
}

func TestFetchBinIdsOnceWithCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"item": {
					"attributes": [
						{
							"attributeCode": "attributes_wasteContainersAssignableWasteContainers",
							"value": [
								"foo",
								"bar",
								"baz"
							]
						}
					]
				}
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, cache}

	client.GetBinIds(PropertyId, Token)
	client.GetBinIds(PropertyId, Token)

	assert.Equal(t, fetches, 1)
}
