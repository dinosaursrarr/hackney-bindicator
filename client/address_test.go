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

func TestWrongPostcodeArea(t *testing.T) {
	client := client.BinsClient{http.Client{}, nil, nil, nil, nil}
	res, err := client.GetAddresses("EH16 5AY", Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "must begin with")
}

func TestNotPostcode(t *testing.T) {
	tests := []string{
		"16 Holyrood Park Road",
		"EH16 5AYX",
		"EH16X5AY",
		"XEH16 5AY",
		"EHE6 5AY",
		"EH16X5AY",
		"EH16ü5AY",
		"ü8 3QQ",
		"E8 3üu",
		"Susan",
	}
	client := client.BinsClient{http.Client{}, nil, nil, nil, nil}

	for _, test := range tests {
		res, err := client.GetAddresses(test, Token)
		assert.Empty(t, res)
		assert.Contains(t, err.Error(), "Not a valid postcode")
	}
}

func TestBadUrlForAddresses(t *testing.T) {
	badUrl, _ := url.Parse("ftp://foo.com")
	client := client.BinsClient{http.Client{}, nil, badUrl, nil, nil}

	res, err := client.GetAddresses(Postcode, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestSetAccessTokenGettingAddresses(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header["Authorization"], "Bearer "+Token)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	client.GetAddresses(Postcode, Token)
}

func TestSetUserAgentGettingAddresses(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["User-Agent"])
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	client.GetAddresses(Postcode, Token)
}

func TestHttpErrorGettingAddresses(t *testing.T) {
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

	res, err := client.GetAddresses(Postcode, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "foo")
}

func TestBadStatusCodeGettingAddresses(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetAddresses(Postcode, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Status code 418")
}

func TestErrorReadingAddresses(t *testing.T) {
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

	res, err := client.GetAddresses(Postcode, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "nope")
}

func TestNoAddressesReturned(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetAddresses(Postcode, Token)

	assert.Empty(t, res)
	assert.Nil(t, err)
}

func TestAcceptPostcodeFormats(t *testing.T) {
	tests := []string{
		"E8 3QQ",
		"E83QQ",
		"e83qq",
		"e8 3qq",
		"E8 3qq",
		"e8 3QQ",
		"e83qQ",
	}
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	for _, test := range tests {
		res, err := client.GetAddresses(test, Token)
		assert.Empty(t, res)
		assert.Nil(t, err)
	}
}

func TestSuccessAddresses(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"itemId": "foo",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "bar"
							}
						]
					}
				]
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := binsClient.GetAddresses(Postcode, Token)

	expected := client.Address{
		Id:   "foo",
		Name: "bar",
	}
	assert.Equal(t, res, []client.Address{expected})
	assert.Nil(t, err)
}

func TestSkipEmptyName(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"itemId": "foo",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
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
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := binsClient.GetAddresses(Postcode, Token)

	assert.Empty(t, res)
	assert.Nil(t, err)
}

func TestSkipEmptyId(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"itemId": "",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "bar"
							}
						]
					}
				]
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := binsClient.GetAddresses(Postcode, Token)

	assert.Empty(t, res)
	assert.Nil(t, err)
}

func TestSuccessFetchTwiceWithoutCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches += 1
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"itemId": "foo",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "bar"
							}
						]
					}
				]
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	binsClient.GetAddresses(Postcode, Token)
	binsClient.GetAddresses(Postcode, Token)

	assert.Equal(t, fetches, 2)
}

func TestSuccessFetchOnceWithCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches += 1
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"itemId": "foo",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "bar"
							}
						]
					}
				]
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil, cache}

	binsClient.GetAddresses(Postcode, Token)
	binsClient.GetAddresses(Postcode, Token)

	assert.Equal(t, fetches, 1)
}

func TestSortByName(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"results": [
					{
						"itemId": "1",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "bar"
							}
						]
					},
					{
						"itemId": "2",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "aaa"
							}
						]
					},
					{
						"itemId": "3",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "ba r"
							}
						]
					},
					{
						"itemId": "4",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "10 Smith Road"
							}
						]
					},
					{
						"itemId": "5",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "9 Smith Road"
							}
						]
					},
					{
						"itemId": "6",
						"attributes": [
							{
								"attributeCode": "attributes_itemsTitle",
								"value": "1 Smith Road"
							}
						]
					}
				]
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	binsClient := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := binsClient.GetAddresses(Postcode, Token)

	expected := []client.Address{
		client.Address{
			Id:   "6",
			Name: "1 Smith Road",
		},
		client.Address{
			Id:   "5",
			Name: "9 Smith Road",
		},
		client.Address{
			Id:   "4",
			Name: "10 Smith Road",
		},
		client.Address{
			Id:   "2",
			Name: "aaa",
		},
		client.Address{
			Id:   "3",
			Name: "ba r",
		},
		client.Address{
			Id:   "1",
			Name: "bar",
		},
	}
	assert.Equal(t, res, expected)
	assert.Nil(t, err)
}
