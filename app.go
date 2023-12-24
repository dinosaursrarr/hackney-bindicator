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
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jonboulle/clockwork"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"

	}

	// Default expiry time of 15 mins, up to 4k cached entries
	cache := expirable.NewLRU[string, interface{}](4096, nil, time.Minute*15)

	httpClient := http.Client{}
	clock := clockwork.NewRealClock()
	apiHost, _ := url.Parse("https://api.uk.alloyapp.io")
	startUrl, _ := url.Parse("https://hackney-waste-pages.azurewebsites.net")
	binsClient := client.BinsClient{httpClient, clock, apiHost, startUrl, cache}

	collectionHandler := handler.CollectionHandler{binsClient, cache}
	addressHandler := handler.AddressHandler{binsClient, cache}

	r := mux.NewRouter()
	r.HandleFunc("/", helloHandler)
	r.HandleFunc("/property/{property_id}", collectionHandler.Handle)
	r.HandleFunc("/addresses/{postcode}", addressHandler.Handle)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
