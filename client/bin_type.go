package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c BinsClient) GetBinType(binId, token string) (string, error) {
	target := c.ApiHost.JoinPath(itemUrl, binId).String()

	if c.Cache != nil {
		if res, found := c.Cache.Get(target); found {
			return res.(string), nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil { // Don't think this can fail
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", token))

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Status code %v fetching types of bins", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// In this case, we want a single-stringed attribute value
	type item struct {
		Item struct {
			Attributes []struct {
				AttributeCode string `json:"attributeCode"`
				Value         string `json:"value"`
			} `json:"attributes"`
		} `json:"item"`
	}

	var data item
	json.Unmarshal(body, &data)
	for _, attribute := range data.Item.Attributes {
		if attribute.AttributeCode != "attributes_itemsSubtitle" {
			continue
		}
		if c.Cache != nil {
			c.Cache.Add(target, attribute.Value)
		}
		return attribute.Value, nil
	}

	return "", errors.New("Bin type not found")
}
