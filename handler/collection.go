package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/gorilla/mux"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/errgroup"
)

type CollectionHandler struct {
	Client client.BinsClient
	Cache  *expirable.LRU[string, interface{}]
}

func (h *CollectionHandler) Handle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	propertyId := vars["property_id"]
	if propertyId == "" {
		http.Error(w, "URL did not include property_id", http.StatusBadRequest)
		return
	}

	if h.Cache != nil {
		if res, found := h.Cache.Get(r.URL.String()); found {
			result := res.(string)
			fmt.Fprintf(w, result)
			return
		}
	}

	binIds, err := h.Client.GetBinIds(propertyId)
	if err != nil {
		if err == client.ErrBadPropertyId {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	g := new(errgroup.Group)
	binTypes := make([]client.BinType, len(binIds.Ids))
	binWorkflowIds := make([]string, len(binIds.Ids))
	var schedulesStarted sync.Map
	var schedules sync.Map
	for i, binId := range binIds.Ids {
		i := i
		binId := binId
		g.Go(func() error {
			binType, err := h.Client.GetBinType(binId)
			if err != nil {
				return err
			}
			binTypes[i] = binType
			return nil
		})
		g.Go(func() error {
			workflowId, err := h.Client.GetBinWorkflowId(binId)
			if err != nil {
				return err
			}
			binWorkflowIds[i] = workflowId
			// Fetch the schedule as soon as we see this ID, but only the first time.
			if _, ok := schedulesStarted.Load(workflowId); ok {
				return nil
			}
			schedulesStarted.Store(workflowId, true)
			schedule, err := h.Client.GetWorkflowSchedule(workflowId)
			if err != nil {
				return err
			}
			schedules.Store(workflowId, schedule)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type bin struct {
		Name           string
		Type           string
		NextCollection time.Time
	}
	type result struct {
		PropertyId string
		Name       string
		Bins       []bin
	}
	var bins []bin
	for i, _ := range binIds.Ids {
		s, ok := schedules.Load(binWorkflowIds[i])
		if !ok {
			continue
		}
		schedule := s.([]time.Time)
		if len(schedule) == 0 {
			continue
		}
		bins = append(bins, bin{
			Name:           binTypes[i].Name,
			Type:           binTypes[i].Type.String(),
			NextCollection: schedule[0],
		})
	}

	resBytes, err := json.Marshal(result{
		PropertyId: propertyId,
		Name:       binIds.Name,
		Bins:       bins,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := string(resBytes)
	if h.Cache != nil {
		h.Cache.Add(r.URL.String(), res)
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, res)
}
