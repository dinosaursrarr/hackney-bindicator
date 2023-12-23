package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c BinsClient) GetBinWorkflowId(binId, token string) (string, error) {
	reqBody := []byte(`{
        "type": "Query",
        "aqs": {
            "properties": {
                "dodiCode": "designs_scheduleCode_5f8d5eac8dae040066c53c8c",
                "attributes": [
                    "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec",
                ],
            },
            "children": [
                {
                    "type": "Equals",
                    "children": [
                        {
                            "type": "Attribute",
                            "properties": {
                                "attributeCode": "attributes_assetGroupsAssets",
                                "path": "root^Template:attributes_wasteRoundsScheduleCode_5f8de8de8dae040066c59dae.Template:attributes_projectsTasks^Live:attributes_tasksAssignableTasks<designInterfaces_assetGroups>",
                                "value": [],
                            },
                        },
                        {
                            "type": "AlloyId",
                            "properties": {
                                "value": [
                                    "` + binId + `",
                                ],
                            },
                        },
                    ],
                },
            ],
        },
    }`)
	target := c.ApiHost.JoinPath(queryUrl)

	cacheKey := target.JoinPath(binId).String()
	if c.Cache != nil {
		if res, found := c.Cache.Get(cacheKey); found {
			return res.(string), nil
		}
	}

	req, err := http.NewRequest(http.MethodPost, target.String(), bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", token))
	req.Header.Add("Content-Type", "application/json")

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
		Results []struct {
			Attributes []struct {
				AttributeCode string `json:"attributeCode"`
				Value         string `json:"value"`
			} `json:"attributes"`
		} `json:"results"`
	}

	var data result
	json.Unmarshal(respBody, &data)
	for _, res := range data.Results {
		for _, attribute := range res.Attributes {
			if attribute.AttributeCode != "attributes_scheduleCodeWorkflowID_5f8dbfdce27d98006789b4ec" {
				continue
			}

			if c.Cache != nil {
				c.Cache.Add(cacheKey, attribute.Value)
			}

			return attribute.Value, nil
		}
	}
	return "", errors.New("Workflow ID not found")
}
