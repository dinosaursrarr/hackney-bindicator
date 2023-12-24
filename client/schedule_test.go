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
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
)

func TestBadWorkflowScheduleUrl(t *testing.T) {
	badUrl, _ := url.Parse("ftp://foo.bar")
	client := client.BinsClient{http.Client{}, nil, badUrl, nil, nil}

	res, err := client.GetWorkflowSchedule(WorkflowId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "unsupported protocol scheme")
}

func TestSetAccessTokenGettingWorkflowSchedule(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header["Authorization"], "Bearer "+Token)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, nil}

	client.GetWorkflowSchedule(WorkflowId, Token)
}

func TestSetUserAgentGettingWorkflowSchedule(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header["User-Agent"])
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, nil}

	client.GetWorkflowSchedule(WorkflowId, Token)
}

func TestHttpErrorGettingWorkflowSchedule(t *testing.T) {
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

	res, err := client.GetWorkflowSchedule(WorkflowId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "foo")
}

func TestBadStatusCodeGettingWorkflowSchedule(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	client := client.BinsClient{http.Client{}, nil, apiUrl, nil, nil}

	res, err := client.GetWorkflowSchedule(WorkflowId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "Status code 418")
}

func TestErrorReadingWorkflowSchedule(t *testing.T) {
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

	res, err := client.GetWorkflowSchedule(WorkflowId, Token)

	assert.Empty(t, res)
	assert.Contains(t, err.Error(), "nope")
}

func TestWorkflowScheduleNotFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, nil}

	res, err := client.GetWorkflowSchedule(WorkflowId, Token)

	assert.Empty(t, res)
	assert.Nil(t, err)
}

func TestEmptyWorkflowScheduleFound(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"workflow": {
					"workflow": {
						"trigger": {
							"dates": [
							]
						}
					}
				}
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, nil}

	res, err := client.GetWorkflowSchedule(WorkflowId, Token)

	assert.Empty(t, res)
	assert.Nil(t, err)
}

func TestSuccessWorkflowSchedule(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"workflow": {
					"workflow": {
						"trigger": {
							"dates": [
								"2023-12-22T13:55:42.123Z",
								"2024-01-05T09:22:31.000Z",
								"2025-07-06T12:00:00.002Z"
							]
						}
					}
				}
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, nil}

	res, err := client.GetWorkflowSchedule(BinId, Token)

	a := time.Date(2023, 12, 22, 0, 0, 0, 0, london)
	b := time.Date(2024, 1, 5, 0, 0, 0, 0, london)
	c := time.Date(2025, 7, 6, 0, 0, 0, 0, london)
	assert.Equal(t, []time.Time{a, b, c}, res)
	assert.Nil(t, err)
}

func TestFilterWorkflowsInPast(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"workflow": {
					"workflow": {
						"trigger": {
							"dates": [
								"2023-12-22T13:55:42.123Z", `+ /* not included */ `
								"2024-01-05T09:22:31.000Z",
								"2025-07-06T12:00:00.002Z"
							]
						}
					}
				}
			}
		`)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2024, 1, 1, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, nil}

	res, err := client.GetWorkflowSchedule(BinId, Token)

	b := time.Date(2024, 1, 5, 0, 0, 0, 0, london)
	c := time.Date(2025, 7, 6, 0, 0, 0, 0, london)
	assert.Equal(t, []time.Time{b, c}, res)
	assert.Nil(t, err)
}

func TestFetchScheduleTwiceWithoutCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"workflow": {
					"workflow": {
						"trigger": {
							"dates": [
								"2023-12-22T13:55:42.123Z"
							]
						}
					}
				}
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, nil}

	client.GetWorkflowSchedule(BinId, Token)
	client.GetWorkflowSchedule(BinId, Token)

	assert.Equal(t, fetches, 2)
}

func TestFetchScheduleOnceWithCache(t *testing.T) {
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			{
				"workflow": {
					"workflow": {
						"trigger": {
							"dates": [
								"2023-12-22T13:55:42.123Z"
							]
						}
					}
				}
			}
		`)
		fetches += 1
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	client := client.BinsClient{http.Client{}, clock, apiUrl, nil, cache}

	client.GetWorkflowSchedule(BinId, Token)
	client.GetWorkflowSchedule(BinId, Token)

	assert.Equal(t, fetches, 1)
}
