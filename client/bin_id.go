package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Want to be able to distinguish this error from other statuses.
var ErrBadPropertyId = errors.New("Status code 400 fetching list of bins")

type BinIds struct {
	Name string
	Ids  []string
}

func (c BinsClient) GetBinIds(propertyId string) (BinIds, error) {
	target := c.ApiHost.JoinPath(binIdUrl, propertyId).String()

	if c.Cache != nil {
		if res, found := c.Cache.Get(target); found {
			return res.(BinIds), nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return BinIds{}, err
	}
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Accept", "application/json")

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
		AddressSummary string `json:"addressSummary"`
		Fields struct {
			Containers string `json:"attributes_wasteContainersAssignableWasteContainers"`
		} `json:"providerSpecificFields"`
	}

	var data item
	json.Unmarshal(body, &data)
	res := BinIds{
		Name: tidy(data.AddressSummary),
		Ids: strings.Split(data.Fields.Containers, ","),
	}
	if len(res.Ids) == 0 {
		return BinIds{}, errors.New("Bin IDs not found for property")
	}
	if c.Cache != nil {
		c.Cache.Add(target, res)
	}
	return res, nil
}
