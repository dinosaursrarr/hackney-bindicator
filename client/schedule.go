package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func (c BinsClient) GetWorkflowSchedule(workflowId, token string) ([]time.Time, error) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		return []time.Time{}, err
	}
	target := c.ApiHost.JoinPath(workflowUrl, workflowId).String()
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return []time.Time{}, err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", token))

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return []time.Time{}, err
	}
	if resp.StatusCode != 200 {
		return []time.Time{}, fmt.Errorf("Status code %v fetching workflow schedule", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []time.Time{}, err
	}

	type workflow struct {
		Workflow struct {
			Workflow struct {
				Trigger struct {
					Dates []time.Time `json:"dates"`
				} `json:"trigger"`
			} `json:"workflow"`
		} `json:"workflow"`
	}

	var data workflow
	json.Unmarshal(body, &data)

	var schedule []time.Time
	now := c.Clock.Now()
	for _, t := range data.Workflow.Workflow.Trigger.Dates {
		if t.Before(now) {
			continue
		}
		date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, london)
		schedule = append(schedule, date)
	}

	return schedule, nil
}
