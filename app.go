package main

import (
	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/dinosaursrarr/hackney-bindicator/handler"

	"embed"
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

//go:embed README.md
var readme []byte

//go:embed static/*
var static embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"

	}

	// Default expiry time of 15 mins, up to 4k cached entries
	cache := expirable.NewLRU[string, interface{}](4096, nil, time.Minute*15)

	httpClient := http.Client{}
	clock := clockwork.NewRealClock()
	apiHost, _ := url.Parse("https://waste-api-hackney-live.ieg4.net/f806d91c-e133-43a6-ba9a-c0ae4f4cccf6")
	binsClient := client.BinsClient{
		HttpClient: httpClient,
		Clock:      clock,
		ApiHost:    apiHost,
		Cache:      cache,
	}

	collectionHandler := handler.CollectionHandler{
		Client: binsClient,
		Cache:  cache,
	}
	addressHandler := handler.AddressHandler{
		Client: binsClient,
		Cache:  cache,
	}
	readmeHandler := handler.MarkdownHandler{
		Markdown: readme,
		Title:    "Hackney Bindicator",
		CssPath:  "static/style.css",
	}

	r := mux.NewRouter()
	r.HandleFunc("/property/{property_id}", collectionHandler.Handle)
	r.HandleFunc("/addresses/{postcode}", addressHandler.Handle)
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(static)))
	r.HandleFunc("/", readmeHandler.Handle)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
