package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Want to be able to distinguish this error from other statuses.
var ErrBadPropertyId = errors.New("Status code 400 fetching list of bins")

type BinIds struct {
	Name string
	Ids  []string
}

func (c BinsClient) GetBinIds(propertyId, token string) (BinIds, error) {
	target := c.ApiHost.JoinPath(itemUrl, propertyId).String()

	if c.Cache != nil {
		if res, found := c.Cache.Get(target); found {
			return res.(BinIds), nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return BinIds{}, err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", token))
	req.Header.Add("User-Agent", userAgent)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return BinIds{}, err
	}
	if resp.StatusCode == 400 {
		return BinIds{}, ErrBadPropertyId
	}
	if resp.StatusCode != 200 {
		return BinIds{}, fmt.Errorf("Status code %v fetching list of bins", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BinIds{}, err
	}

	// In this case, we want an attribute with a list of string values
	type item struct {
		Item struct {
			Attributes []struct {
				AttributeCode string          `json:"attributeCode"`
				Value         json.RawMessage `json:"value"`
			} `json:"attributes"`
		} `json:"item"`
	}

	var data item
	json.Unmarshal(body, &data)
	var res BinIds
	for _, attribute := range data.Item.Attributes {
		if attribute.AttributeCode == "attributes_itemsTitle" {
			var val string
			json.Unmarshal(attribute.Value, &val)
			res.Name = tidy(val)
		}
		if attribute.AttributeCode == "attributes_wasteContainersAssignableWasteContainers" {
			json.Unmarshal(attribute.Value, &res.Ids)
		}
	}
	if len(res.Ids) == 0 {
		return BinIds{}, errors.New("Bin IDs not found for property")
	}
	if c.Cache != nil {
		c.Cache.Add(target, res)
	}
	return res, nil
}
