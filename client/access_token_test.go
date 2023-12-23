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

    "github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/stretchr/testify/assert"
)

type fakeRoundTripper struct {
    Fn func(*http.Request)(*http.Response, error)
}
func (frt fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    return frt.Fn(req)
}

func TestSuccessAccessToken(t *testing.T) {
    svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "<script>ALLOY_APP_TOKEN: \"foo\"</script>")
    }))
    defer svr.Close()
    startUrl, _ := url.Parse(svr.URL)
    client := client.BinsClient{http.Client{}, nil, nil, startUrl}

    res, err := client.GetAccessToken()

    assert.Equal(t, res, "foo")
    assert.Nil(t, err)
}

func TestBadUrlForAccessToken(t *testing.T) {
    badUrl, _ := url.Parse("ftp://foo.com")
    client := client.BinsClient{http.Client{}, nil, nil, badUrl}

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
    client := client.BinsClient{httpClient, nil, nil, startUrl}

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
    client := client.BinsClient{http.Client{}, nil, nil, startUrl}

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
                    Body: ioutil.NopCloser(iotest.ErrReader(errors.New("nope"))),
                }, nil
            },
        },
    }
    client := client.BinsClient{httpClient, nil, nil, startUrl}

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
    client := client.BinsClient{http.Client{}, nil, nil, startUrl}

    res, err := client.GetAccessToken()

    assert.Equal(t, res, "")
    assert.Contains(t, err.Error(), "Could not find")
}