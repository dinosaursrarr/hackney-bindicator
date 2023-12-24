package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/dinosaursrarr/hackney-bindicator/handler"
	"github.com/gorilla/mux"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
)

const Postcode = "E8 1EA" // Hackney Town Hall

const AddressJsonResponse = `
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
`

func TestNoPostcode(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"postcode": "",
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, &url.URL{}, &url.URL{}, nil}
	handler := handler.AddressHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	assert.Contains(t, w.Body.String(), "include postcode")
}

func TestAddressCannotGetAccessToken(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No access token", http.StatusTeapot)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"postcode": Postcode,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, &url.URL{}, startUrl, nil}
	handler := handler.AddressHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching access token")
}

func TestNonHackneyPostcode(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"postcode": "EH16 5AY",
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, &url.URL{}, startUrl, nil}
	handler := handler.AddressHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	assert.Contains(t, w.Body.String(), "Hackney postcodes")
}

func TestInvalidPostcode(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"postcode": "E8 1EAXXX",
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, &url.URL{}, startUrl, nil}
	handler := handler.AddressHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	assert.Contains(t, w.Body.String(), "Not a valid postcode")
}

func TestOtherErrorGettingAddresses(t *testing.T) {
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
		"postcode": Postcode,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.AddressHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Contains(t, w.Body.String(), "fetching addresses")
}

func TestSuccess(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, AddressJsonResponse)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w := httptest.NewRecorder()
	vars := map[string]string{
		"postcode": Postcode,
	}
	r = mux.SetURLVars(r, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.AddressHandler{client, nil}

	handler.Handle(w, r)

	assert.Equal(t, w.Code, http.StatusOK)
	assert.JSONEq(t, w.Body.String(), `
		[
			{
				"Id": "foo",
				"Name": "bar"
			}
		]`)
}

func TestFetchAddressesTwiceWithoutCache(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches += 1
		fmt.Fprintf(w, AddressJsonResponse)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r1, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w1 := httptest.NewRecorder()
	r2, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w2 := httptest.NewRecorder()
	vars := map[string]string{
		"postcode": Postcode,
	}
	r1 = mux.SetURLVars(r1, vars)
	r2 = mux.SetURLVars(r2, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	handler := handler.AddressHandler{client, nil}

	handler.Handle(w1, r1)
	handler.Handle(w2, r2)

	assert.Equal(t, fetches, 2)
}

func TestFetchAddressesOnceWithCache(t *testing.T) {
	startSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, StartPage)
	}))
	startUrl, _ := url.Parse(startSvr.URL)
	fetches := 0
	apiSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches += 1
		fmt.Fprintf(w, AddressJsonResponse)
	}))
	defer apiSvr.Close()
	apiUrl, _ := url.Parse(apiSvr.URL)
	r1, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w1 := httptest.NewRecorder()
	r2, _ := http.NewRequest(http.MethodGet, RequestUrl, nil)
	w2 := httptest.NewRecorder()
	vars := map[string]string{
		"postcode": Postcode,
	}
	r1 = mux.SetURLVars(r1, vars)
	r2 = mux.SetURLVars(r2, vars)
	httpClient := http.Client{}
	clock := clockwork.NewFakeClock()
	client := client.BinsClient{httpClient, clock, apiUrl, startUrl, nil}
	cache := expirable.NewLRU[string, interface{}](1024, nil, time.Minute*10)
	handler := handler.AddressHandler{client, cache}

	handler.Handle(w1, r1)
	handler.Handle(w2, r2)

	assert.Equal(t, fetches, 1)
}
