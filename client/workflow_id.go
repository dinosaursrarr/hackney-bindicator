package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c BinsClient) GetBinWorkflowId(binId string) (string, error) {
	target := c.ApiHost.JoinPath(workflowIdUrl, binId).String()

	if c.Cache != nil {
		if res, found := c.Cache.Get(target); found {
			return res.(string), nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", userAgent)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Status code %v fetching workflows of bins", resp.StatusCode)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type result struct {
		IDs []string `json:"scheduleCodeWorkflowIDs"`
	}

	var data result
	json.Unmarshal(respBody, &data)

	if len(data.IDs) == 0 {
		return "", errors.New("Workflow IDs not found")
	}
	if data.IDs[0] == "" {
		return "", errors.New("Workflow ID not found")
	}

	if c.Cache != nil {
		c.Cache.Add(target, data.IDs[0])
	}
	return data.IDs[0], nil
}
