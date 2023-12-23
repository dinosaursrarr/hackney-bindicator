package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dinosaursrarr/hackney-bindicator/client"
	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/errgroup"
)

type CollectionHandler struct {
	Client client.BinsClient
	Cache  *cache.Cache
}

func (h *CollectionHandler) Handle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	propertyId := vars["property_id"]
	if propertyId == "" {
		http.Error(w, "URL did not include property_id", http.StatusBadRequest)
		return
	}

	buf := new(bytes.Buffer)
	if err := r.Write(buf); err != nil {
		http.Error(w, "Could not serialise request", http.StatusInternalServerError)
		return
	}
	cacheKey := buf.String()
	if h.Cache != nil {
		if res, found := h.Cache.Get(cacheKey); found {
			result := res.(string)
			fmt.Fprintf(w, result)
			return
		}
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

	g := new(errgroup.Group)
	binTypes := make([]string, len(binIds))
	binWorkflowIds := make([]string, len(binIds))
	var schedulesStarted sync.Map
	var schedules sync.Map
	for i, binId := range binIds {
		i := i
		binId := binId
		g.Go(func() error {
			binType, err := h.Client.GetBinType(binId, token)
			if err != nil {
				return err
			}
			binTypes[i] = binType
			return nil
		})
		g.Go(func() error {
			workflowId, err := h.Client.GetBinWorkflowId(binId, token)
			if err != nil {
				return err
			}
			binWorkflowIds[i] = workflowId
			// Fetch the schedule as soon as we see this ID, but only the first time.
			if _, ok := schedulesStarted.Load(workflowId); ok {
				return nil
			}
			schedulesStarted.Store(workflowId, true)
			schedule, err := h.Client.GetWorkflowSchedule(workflowId, token)
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
		NextCollection time.Time
	}
	type result struct {
		PropertyId string
		Bins       []bin
	}
	var bins []bin
	for i, _ := range binIds {
		s, ok := schedules.Load(binWorkflowIds[i])
		if !ok {
			continue
		}
		schedule := s.([]time.Time)
		if len(schedule) == 0 {
			continue
		}
		bins = append(bins, bin{
			Name:           binTypes[i],
			NextCollection: schedule[0],
		})
	}

	resBytes, err := json.Marshal(result{PropertyId: propertyId, Bins: bins})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := string(resBytes)
	if h.Cache != nil {
		h.Cache.Set(cacheKey, res, cache.DefaultExpiration)
	}
	fmt.Fprintf(w, res)
}
