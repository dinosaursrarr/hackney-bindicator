package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/gorilla/mux"
)

type CollectionHandler struct {
	Client client.BinsClient
}

func (h *CollectionHandler) Handle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	propertyId := vars["property_id"]
	if propertyId == "" {
		http.Error(w, "URL did not include property_id", http.StatusBadRequest)
		return
	}

	token, err := h.Client.GetAccessToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	binIds, err := h.Client.GetBinIds(propertyId, token)
	if err != nil {
		if err == client.ErrBadPropertyId {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	binTypes := make([]string, len(binIds))
	binWorkflowIds := make([]string, len(binIds))
	for i, binId := range binIds {
		binType, err := h.Client.GetBinType(binId, token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		binTypes[i] = binType

		workflowId, err := h.Client.GetBinWorkflowId(binId, token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		binWorkflowIds[i] = workflowId
	}

	schedules := map[string][]time.Time{}
	for _, workflowId := range binWorkflowIds {
		schedules[workflowId] = []time.Time{}
	}

	for workflowId, _ := range schedules {
		workflowId := workflowId
		schedule, err := h.Client.GetWorkflowSchedule(workflowId, token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		schedules[workflowId] = schedule
	}

	type bin struct {
		Name           string
		NextCollection time.Time
	}
	type result struct {
		PropertyId string
		Bins       []bin
	}
	var bins []bin
	for i, _ := range binIds {
		if len(schedules[binWorkflowIds[i]]) == 0 {
			continue
		}
		bins = append(bins, bin{
			Name:           binTypes[i],
			NextCollection: schedules[binWorkflowIds[i]][0],
		})
	}

	res, err := json.Marshal(result{PropertyId: propertyId, Bins: bins})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(res))
}
