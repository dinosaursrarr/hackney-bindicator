package handler_test

import (
	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/dinosaursrarr/hackney-bindicator/handler"

	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	_ "time/tzdata"

	"github.com/gorilla/mux"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
)

const PropertyId = "property_id"
const BinId1 = "bin1"
const BinId2 = "bin2"
const BinIdJsonResponse = `
	{
		"item": {
			"attributes": [
				{
					"attributeCode": "attributes_wasteContainersAssignableWasteContainers",
					"value": [
						"` + BinId1 + `",
						"` + BinId2 + `"
					]
				}
			]
		}
	}
`
const Bin1Type = "Garbage can"
const Bin1RefuseType = "5f96b6f8d1f4f500660f3058"
const Bin1TypeJsonResponse = `
	{
		"item": {
			"attributes": [
				{
					"attributeCode": "attributes_itemsSubtitle",
					"value": "` + Bin1Type + `"
				},
				{
					"attributeCode": "attributes_wasteContainersType",
					"value": [
						"` + Bin1RefuseType + `"
					]
				}
			]
		}
	}
`
const Bin2Type = "Dumpster"
const Bin2TypeJsonResponse = `
	{
		"item": {
			"attributes": [
				{
					"attributeCode": "attributes_itemsSubtitle",
					"value": "` + Bin2Type + `"
				}
			]
		}
	}
`
const WorkflowId1 = "workflow1"
const Bin1WorkflowIdJsonResponse = `
	{
		"results": [
			{
				"attributes": [
					{
						"attributeCode": "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec",
						"value": "` + WorkflowId1 + `"
					}
				]
			}
		]
	}
`
const WorkflowId2 = "workflow2"
const Bin2WorkflowIdJsonResponse = `
	{
		"results": [
			{
				"attributes": [
					{
						"attributeCode": "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec",
						"value": "` + WorkflowId2 + `"
					}
				]
			}
		]
	}
`
const Workflow1ScheduleJsonResponse = `
	{
		"workflow": {
			"workflow": {
				"trigger": {
					"dates": [
						"2023-12-01T13:55:42.123Z",
						"2024-01-01T09:22:31.000Z",
						"2025-07-01T12:00:00.002Z"
					]
				}
			}
		}
	}
`
const Workflow2ScheduleJsonResponse = `
	{
		"workflow": {
			"workflow": {
				"trigger": {
					"dates": [
						"2023-12-02T13:55:42.123Z",
						"2024-01-02T09:22:31.000Z",
						"2025-07-02T12:00:00.002Z"
					]
				}
			}
		}
	}
`

func TestNoPropertyId(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": "",
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, &url.URL{}, &url.URL{}, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	assert.Contains(t, w.Body.String(), "include property_id")
}

func TestCannotGetAccessToken(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No access token", http.StatusTeapot)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, &url.URL{}, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching access token")
}

func TestClientErrorGettingBinIds(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	assert.Contains(t, w.Body.String(), "fetching list of bins")
}

func TestServerErrorGettingBinIds(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching list of bins")
}

func TestErrorGettingBinTypes(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			http.Error(w, "can't get bin type", http.StatusTeapot)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin2WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "fetching types of bins")
}

func TestErrorGettingBinWorkflowIds(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			http.Error(w, "can't get bin workflow id", http.StatusTeapot)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin2WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching workflows of bins")
}

func TestErrorGettingWorkflowSchedules(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/workflow/") {
			http.Error(w, "nope", http.StatusTeapot)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/workflow/") {
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin2WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching workflow schedule")
}

func TestNextCollectionDateForEachBin(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/workflow/") {
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/workflow/") {
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin2WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.JSONEq(t, w.Body.String(), `
		{
			"PropertyId": "property_id",
			"Bins": [
				{
					"Name": "Garbage can",
					"Type": "garden",
					"NextCollection": "2024-01-01T00:00:00Z"
				},
				{
					"Name": "Dumpster",
					"Type": "unknown",
					"NextCollection": "2024-01-02T00:00:00Z"
				}
			]
		}`)
	assert.Contains(t, w.Header(), "Content-Type")
}

func TestOnlyFetchEachUniqueWorkflowScheduleOnce(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	fetchCount := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/workflow/") {
			fetchCount += 1
			assert.Less(t, fetchCount, 2)
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			// Return same workflow ID so it should only be fetched once
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.JSONEq(t, w.Body.String(), `
		{
			"PropertyId": "property_id",
			"Bins": [
				{
					"Name": "Garbage can",
					"Type": "garden",
					"NextCollection": "2024-01-01T00:00:00Z"
				},
				{
					"Name": "Dumpster",
					"Type": "unknown",
					"NextCollection": "2024-01-01T00:00:00Z"
				}
			]
		}`)
}

func TestSkipBinWithNoNextCollection(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/workflow/") {
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/workflow/") {
			// All dates are in the past, so bin is skipped
			fmt.Fprintf(w, `{
				"workflow": {
					"workflow": {
						"trigger": {
							"dates": [
								"1998-12-01T13:55:42.123Z",
								"1999-01-02T09:22:31.000Z",
								"1999-07-02T12:00:00.002Z"
							]
						}
					}
				}
			}`)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			fmt.Fprintf(w, Bin2WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.JSONEq(t, w.Body.String(), `
		{
			"PropertyId": "property_id",
			"Bins": [
				{
					"Name": "Garbage can",
					"Type": "garden",
					"NextCollection": "2024-01-01T00:00:00Z"
				}
			]
		}`)
}

func TestFetchTwiceWithoutCache(t *testing.T) {
	fetches := make(map[string]int)
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches[r.URL.String()] += 1
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/workflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/workflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			fetches[body] += 1
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			fetches[body] += 1
			fmt.Fprintf(w, Bin2WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r1, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w1 := httptest.NewRecorder()
	r2, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w2 := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r1 = mux.SetURLVars(r1, vars)
	r2 = mux.SetURLVars(r2, vars)
	httpClient := http.Client{}
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w1, r1)
	handler.Handle(w2, r2)

	assert.Equal(t, w1.Code, http.StatusOK)
	assert.Equal(t, w2.Code, http.StatusOK)

	for _, v := range fetches {
		assert.Equal(t, v, 2)
	}
}

func TestFetchOnceWithCache(t *testing.T) {
	fetches := make(map[string]int)
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches[r.URL.String()] += 1
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/item/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/item/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/workflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/workflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		b, _ := ioutil.ReadAll(r.Body)
		body := string(b)
		if strings.Contains(body, BinId1) && strings.Contains(r.URL.String(), "/query") {
			fetches[body] += 1
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(body, BinId2) && strings.Contains(r.URL.String(), "/query") {
			fetches[body] += 1
			fmt.Fprintf(w, Bin2WorkflowIdJsonResponse)
		}
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r1, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w1 := httptest.NewRecorder()
	r2, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w2 := httptest.NewRecorder()
	vars := map[string]string{
		"property_id": PropertyId,
	}
	r1 = mux.SetURLVars(r1, vars)
	r2 = mux.SetURLVars(r2, vars)
	httpClient := http.Client{}
	london, _ := time.LoadLocation("Europe/London")
	now := time.Date(2023, 12, 15, 3, 19, 46, 72, london)
	clock := clockwork.NewFakeClockAt(now)
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	handler := handler.CollectionHandler{client, cache}

	handler.Handle(w1, r1)
	handler.Handle(w2, r2)

	assert.Equal(t, w1.Code, http.StatusOK)
	assert.Equal(t, w2.Code, http.StatusOK)

	for _, v := range fetches {
		assert.Equal(t, v, 1)
	}
}
