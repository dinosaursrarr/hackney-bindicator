package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/patrickmn/go-cache"
)

// Want to be able to distinguish this error from other statuses.
var ErrBadPropertyId = errors.New("Status code 400 fetching list of bins")

func (c BinsClient) GetBinIds(propertyId, token string) ([]string, error) {
	target := c.ApiHost.JoinPath(itemUrl, propertyId).String()

	if c.Cache != nil {
		if res, found := c.Cache.Get(target); found {
			return res.([]string), nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return []string{}, err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", token))

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return []string{}, err
	}
	if resp.StatusCode == 400 {
		return []string{}, ErrBadPropertyId
	}
	if resp.StatusCode != 200 {
		return []string{}, fmt.Errorf("Status code %v fetching list of bins", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}

	// In this case, we want an attribute with a list of string values
	type item struct {
		Item struct {
			Attributes []struct {
				AttributeCode string   `json:"attributeCode"`
				Value         []string `json:"value"`
			} `json:"attributes"`
		} `json:"item"`
	}

	var data item
	json.Unmarshal(body, &data)

	for _, attribute := range data.Item.Attributes {
		if attribute.AttributeCode != "attributes_wasteContainersAssignableWasteContainers" {
			continue
		}
		if c.Cache != nil {
			c.Cache.Set(target, attribute.Value, cache.DefaultExpiration)
		}
		return attribute.Value, nil
	}

	return []string{}, errors.New("Bin IDs not found for property")
}
