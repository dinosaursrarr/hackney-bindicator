package handler_test

import (
	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/dinosaursrarr/hackney-bindicator/handler"

	"fmt"
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
const Address = "  29   ACACIA  AVENUE "
const BinId1 = "bin1"
const BinId2 = "bin2"
const BinIdJsonResponse = `
	{
		"addressSummary": "` + Address + `",
		"providerSpecificFields": {
			"attributes_wasteContainersAssignableWasteContainers": "` + BinId1 + `,` + BinId2 + `"
		}
	}
`
const Bin1Type = "Garbage can"
const Bin1RefuseType = "5f96b6f8d1f4f500660f3058"
const Bin1TypeJsonResponse = `
	{
		"subTitle": "` + Bin1Type + `",
		"binType": "` + Bin1RefuseType + `"
	}
`
const Bin2Type = "Dumpster"
const Bin2TypeJsonResponse = `
	{
		"subTitle": "` + Bin2Type + `"
	}
`
const WorkflowId1 = "workflow1"
const Bin1WorkflowIdJsonResponse = `
	{
		"scheduleCodeWorkflowID": "` + WorkflowId1 + `"
	}
`
const WorkflowId2 = "workflow2"
const Bin2WorkflowIdJsonResponse = `
	{
		"scheduleCodeWorkflowID": "` + WorkflowId2 + `"
	}
`
const Workflow1ScheduleJsonResponse = `
	{
		"trigger": {
			"dates": [
				"2023-12-01T13:55:42.123Z",
				"2024-01-01T09:22:31.000Z",
				"2025-07-01T12:00:00.002Z"
			]
		}
	}
`
const Workflow2ScheduleJsonResponse = `
	{
		"trigger": {
			"dates": [
				"2023-12-02T13:55:42.123Z",
				"2024-01-02T09:22:31.000Z",
				"2025-07-02T12:00:00.002Z"
			]
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
	client := client.BinsClient{httpClient, clock, &url.URL{}, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	assert.Contains(t, w.Body.String(), "include property_id")
}

func TestClientErrorGettingBinIds(t *testing.T) {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	assert.Contains(t, w.Body.String(), "fetching list of bins")
}

func TestServerErrorGettingBinIds(t *testing.T) {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching list of bins")
}

func TestErrorGettingBinTypes(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			http.Error(w, "can't get bin type", http.StatusTeapot)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "fetching types of bins")
}

func TestErrorGettingBinWorkflowIds(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			http.Error(w, "can't get bin workflow id", http.StatusTeapot)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching workflows of bins")
}

func TestErrorGettingWorkflowSchedules(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/getworkflow/") {
			http.Error(w, "nope", http.StatusTeapot)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching workflow schedule")
}

func TestNextCollectionDateForEachBin(t *testing.T) {
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.JSONEq(t, w.Body.String(), `
		{
			"PropertyId": "property_id",
			"Name": "29 ACACIA AVENUE",
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
	fetchCount := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fetchCount += 1
			assert.Less(t, fetchCount, 2)
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.JSONEq(t, w.Body.String(), `
		{
			"PropertyId": "property_id",
			"Name": "29 ACACIA AVENUE",
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
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/getworkflow/") {
			// All dates are in the past, so bin is skipped
			fmt.Fprintf(w, `{
				"trigger": {
					"dates": [
						"1998-12-01T13:55:42.123Z",
						"1999-01-02T09:22:31.000Z",
						"1999-07-02T12:00:00.002Z"
					]
				}
			}`)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
	handler := handler.CollectionHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.JSONEq(t, w.Body.String(), `
		{
			"PropertyId": "property_id",
			"Name": "29 ACACIA AVENUE",
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
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			fetches[BinId1] += 1
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
			fetches[BinId2] += 1
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
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
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), PropertyId) {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, BinIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getbin/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin1TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getbin/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Bin2TypeJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId1) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow1ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), WorkflowId2) && strings.Contains(r.URL.String(), "/getworkflow/") {
			fetches[r.URL.String()] += 1
			fmt.Fprintf(w, Workflow2ScheduleJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId1) && strings.Contains(r.URL.String(), "/getcollection/") {
			fetches[BinId1] += 1
			fmt.Fprintf(w, Bin1WorkflowIdJsonResponse)
		}
		if strings.Contains(r.URL.String(), BinId2) && strings.Contains(r.URL.String(), "/getcollection/") {
			fetches[BinId2] += 1
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
	client := client.BinsClient{httpClient, clock, apiUrl, nil}
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
