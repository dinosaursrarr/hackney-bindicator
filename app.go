package main

import (
	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/dinosaursrarr/hackney-bindicator/handler"

	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	_ "time/tzdata"

	"github.com/gorilla/mux"
	"github.com/jonboulle/clockwork"
	"github.com/patrickmn/go-cache"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"

	}

	httpClient := http.Client{}
	clock := clockwork.NewRealClock()
	apiHost, _ := url.Parse("https://api.uk.alloyapp.io")
	startUrl, _ := url.Parse("https://hackney-waste-pages.azurewebsites.net")
	binsClient := client.BinsClient{httpClient, clock, apiHost, startUrl}

	// Default expiry time of 15 mins, purge expired items after 30 mins
	cache := cache.New(15*time.Minute, 30*time.Minute)

	handler := handler.CollectionHandler{binsClient, cache}

	r := mux.NewRouter()
	r.HandleFunc("/", helloHandler)
	r.HandleFunc("/property/{property_id}", handler.Handle)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
