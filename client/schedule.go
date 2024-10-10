package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func (c BinsClient) GetWorkflowSchedule(workflowId string) ([]time.Time, error) {
	target := c.ApiHost.JoinPath(scheduleUrl, workflowId).String()

	if c.Cache != nil {
		if res, found := c.Cache.Get(target); found {
			return res.([]time.Time), nil
		}
	}

	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		return []time.Time{}, err
	}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return []time.Time{}, err
	}
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Accept", "application/json")

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
		Trigger struct {
			Dates []time.Time `json:"dates"`
		} `json:"trigger"`
	}

	var data workflow
	json.Unmarshal(body, &data)

	var schedule []time.Time
	now := c.Clock.Now()
	for _, t := range data.Trigger.Dates {
		if t.Before(now) {
			continue
		}
		date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, london)
		schedule = append(schedule, date)
	}

	if c.Cache != nil {
		c.Cache.Add(target, schedule)
	}

	return schedule, nil
}
