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

func TestSuccessAccessToken(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<script>ALLOY_APP_TOKEN: \"foo\"</script>")
	}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	client := client.BinsClient{http.Client{}, nil, nil, startUrl, nil}

	res, err := client.GetAccessToken()

	assert.Equal(t, res, "foo")
	assert.Nil(t, err)
}

func TestSetUserAgentGettingAccessToken(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["User-Agent"])
	}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	client := client.BinsClient{http.Client{}, nil, nil, startUrl, nil}

	client.GetAccessToken()
}

func TestFetchAccessTokenTwiceWithoutCache(t *testing.T) {
	fetches := 0
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<script>ALLOY_APP_TOKEN: \"foo\"</script>")
		fetches += 1
	}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	client := client.BinsClient{http.Client{}, nil, nil, startUrl, nil}

	client.GetAccessToken()
	client.GetAccessToken()

	assert.Equal(t, fetches, 2)
}

func TestFetchAccessTokenOnceWithCache(t *testing.T) {
	fetches := 0
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<script>ALLOY_APP_TOKEN: \"foo\"</script>")
		fetches += 1
	}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	client := client.BinsClient{http.Client{}, nil, nil, startUrl, cache}

	client.GetAccessToken()
	client.GetAccessToken()

	assert.Equal(t, fetches, 1)
}

func TestBadUrlForAccessToken(t *testing.T) {
	badUrl, _ := url.Parse("ftp://foo.com")
	client := client.BinsClient{http.Client{}, nil, nil, badUrl, nil}

	res, err := client.GetAccessToken()

	assert.Equal(t, res, "")
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestHttpErrorGettingAccessToken(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	httpClient := http.Client{
		Transport: fakeRoundTripper{
			Fn: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("foo")
			},
		},
	}
	client := client.BinsClient{httpClient, nil, nil, startUrl, nil}

	res, err := client.GetAccessToken()

	assert.Equal(t, res, "")
	assert.Contains(t, err.Error(), "foo")
}

func TestBadStatusCodeGettingAccessToken(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	client := client.BinsClient{http.Client{}, nil, nil, startUrl, nil}

	res, err := client.GetAccessToken()

	assert.Equal(t, res, "")
	assert.Contains(t, err.Error(), "Status code")
}

func TestErrorReadingGettingAccessToken(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	httpClient := http.Client{
		Transport: fakeRoundTripper{
			Fn: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(iotest.ErrReader(errors.New("nope"))),
				}, nil
			},
		},
	}
	client := client.BinsClient{httpClient, nil, nil, startUrl, nil}

	res, err := client.GetAccessToken()

	assert.Equal(t, res, "")
	assert.Contains(t, err.Error(), "nope")
}

func TestCannotFindAccessToken(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	}))
	defer svr.Close()
	startUrl, _ := url.Parse(svr.URL)
	client := client.BinsClient{http.Client{}, nil, nil, startUrl, nil}

	res, err := client.GetAccessToken()

	assert.Equal(t, res, "")
	assert.Contains(t, err.Error(), "Could not find")
}
