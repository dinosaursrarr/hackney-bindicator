package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/gorilla/mux"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

type AddressHandler struct {
	Client client.BinsClient
	Cache  *expirable.LRU[string, interface{}]
}

func (h *AddressHandler) Handle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postcode := vars["postcode"]
	if postcode == "" {
		http.Error(w, "URL did not include postcode", http.StatusBadRequest)
		return
	}

	if h.Cache != nil {
		if res, found := h.Cache.Get(r.URL.String()); found {
			result := res.(string)
			io.WriteString(w, result)
			return
		}
	}

	addresses, err := h.Client.GetAddresses(postcode)
	if err == client.NotHackneyErr {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err == client.InvalidPostcodeErr {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resBytes, err := json.Marshal(addresses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := string(resBytes)
	if h.Cache != nil {
		h.Cache.Add(r.URL.String(), res)
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, res)
}
