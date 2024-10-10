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
	client := client.BinsClient{http.Client{}, nil, badUrl, nil}

	res, err := client.GetBinIds(PropertyId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestSetUserAgentGettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["User-Agent"])
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	client.GetBinIds(PropertyId)
}

func TestSetAcceptGettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["Accept"])
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	client.GetBinIds(PropertyId)
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
	client := client.BinsClient{httpClient, nil, apiUrl, nil}

	res, err := client.GetBinIds(PropertyId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "foo")
}

func TestStatusCode400GettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := client.GetBinIds(PropertyId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Status code 400")
}

func TestBadStatusCodeGettingBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := client.GetBinIds(PropertyId)

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
	client := client.BinsClient{httpClient, nil, apiUrl, nil}

	res, err := client.GetBinIds(PropertyId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "nope")
}

func TestBinIdsNotFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := client.GetBinIds(PropertyId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Bin IDs not found")
}

func TestNoBinIdsFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"addressSummary": "foo",
				"providerSpecificFields": {
					"attributes_wasteContainersAssignableWasteContainers": ""
				}
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := binsClient.GetBinIds(PropertyId)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Bin IDs not found")
}

func TestSuccessBinIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"addressSummary": "foo",
				"providerSpecificFields": {
					"attributes_wasteContainersAssignableWasteContainers": "foo,bar,baz"
				}
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	res, err := binsClient.GetBinIds(PropertyId)

	assert.Equal(t, res, client.BinIds{
		Name: "foo",
		Ids:  []string{"foo", "bar", "baz"},
	})
	assert.Nil(t, err)
}

func TestFetchBinIdsTwiceWithoutCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"addressSummary": "foo",
				"providerSpecificFields": {
					"attributes_wasteContainersAssignableWasteContainers": "foo,bar,baz"
				}
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil}

	client.GetBinIds(PropertyId)
	client.GetBinIds(PropertyId)

	assert.Equal(t, fetches, 2)
}

func TestFetchBinIdsOnceWithCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"addressSummary": "foo",
				"providerSpecificFields": {
					"attributes_wasteContainersAssignableWasteContainers": "foo,bar,baz"
				}
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	client := client.BinsClient{http.Client{}, nil, apiUrl, cache}

	client.GetBinIds(PropertyId)
	client.GetBinIds(PropertyId)

	assert.Equal(t, fetches, 1)
}
