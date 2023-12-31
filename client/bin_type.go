package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type RefuseType int

const (
	UndefinedRefuseType RefuseType = iota
	Food
	Recycling
	Garden
	Rubbish
)

func (r RefuseType) String() string {
	switch r {
	case Food:
		return "food"
	case Recycling:
		return "recycling"
	case Garden:
		return "garden"
	case Rubbish:
		return "rubbish"
	}
	return "unknown"
}

type BinType struct {
	Name string
	Type RefuseType
}

func extractType(type_id string) RefuseType {
	// This is business logic that I've added. Probably the most
	// fragile part of this whole app.
	if type_id == "5f96b455e36673006420c529" {
		return Food // Food Caddy (Small)
	}
	if type_id == "5f96b4a7d1f4f500660f2cde" {
		return Food // Food Caddy (Large)
	}
	if type_id == "5f89bea126b55500675f4d08" {
		return Recycling // Recycling Sack
	}
	if type_id == "5f96b7733278d10067b889e4" {
		return Recycling // Recycling Reusable Bag (Estate)
	}
	if type_id == "5f96b6523278d10067b88883" {
		return Garden // Garden Waste Reusable Bag
	}
	if type_id == "5f96b6f8d1f4f500660f3058" {
		return Garden // Compostable Liners
	}
	if type_id == "5f96b596e36673006420c665" {
		return Garden // Garden Waste Bin
	}
	if type_id == "5f96b7dde6d6ef00671d1a04" {
		return Garden // Garden Waste Key
	}
	if type_id == "5f89be840de3b800682a1ce6" {
		return Rubbish // Refuse Sack
	}
	if type_id == "5f96b8d0e36673006420c9ed" {
		return Rubbish // Dustbin 90 ltrs x2
	}
	if type_id == "5f96b8fce36673006420ca1f" {
		return Rubbish // Wheeled Bin (180ltr)
	}
	if type_id == "600ae93423debf006583d078" {
		return Rubbish // Dustbin 90 ltrs x1
	}
	if type_id == "619f87d15c9f9c016ce81494" {
		return Rubbish // Refuse Eurobin (1100 litre) Trade Waste Only
	}
	return UndefinedRefuseType
}

func (c BinsClient) GetBinType(binId, token string) (BinType, error) {
	target := c.ApiHost.JoinPath(itemUrl, binId).String()

	if c.Cache != nil {
		if res, found := c.Cache.Get(target); found {
			return res.(BinType), nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil { // Don't think this can fail
		return BinType{}, err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", token))
	req.Header.Add("User-Agent", userAgent)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return BinType{}, err
	}
	if resp.StatusCode != 200 {
		return BinType{}, fmt.Errorf("Status code %v fetching types of bins", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BinType{}, err
	}

	// In this case, we want a single-stringed attribute value
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

	var name string
	var refuseType RefuseType

	for _, attribute := range data.Item.Attributes {
		if attribute.AttributeCode == "attributes_itemsSubtitle" {
			json.Unmarshal(attribute.Value, &name)
			continue
		}
		if attribute.AttributeCode == "attributes_wasteContainersType" {
			var val []string
			json.Unmarshal(attribute.Value, &val)
			if len(val) == 0 {
				continue
			}
			refuseType = extractType(val[0])
			continue
		}
	}

	if name == "" && refuseType == UndefinedRefuseType {
		return BinType{}, errors.New("Bin type not found")
	}

	res := BinType{
		Name: name,
		Type: refuseType,
	}

	if c.Cache != nil {
		c.Cache.Add(target, res)
	}
	return res, nil
}
