package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"facette.io/natsort"
)

type Address struct {
	Id   string
	Name string
}

var hackneyPostcodes = []string{
	"E1",
	"E2",
	"E5",
	"E8",
	"E9",
	"E10",
	"E15",
	"E20",
	"N1",
	"N4",
	"N5",
	"N16",
}

var space = regexp.MustCompile(`\s+`)
var postcode = regexp.MustCompile(`^(?P<outer>[A-Z]{1,2}[0-9][A-Z0-9]?) ?(?P<inner>[0-9][A-Z]{2})$`)

var NotHackneyErr = errors.New("Hackney postcodes must begin with one of " + strings.Join(hackneyPostcodes, ", "))
var InvalidPostcodeErr = errors.New("Not a valid postcode")

func canonicalize(s string) (string, error) {
	tidy := strings.ToUpper(space.ReplaceAllString(strings.TrimSpace(s), " "))
	if !postcode.MatchString(tidy) {
		return "", InvalidPostcodeErr
	}
	outer := postcode.ReplaceAllString(tidy, "${outer}")
	if !slices.Contains(hackneyPostcodes, outer) {
		return "", NotHackneyErr
	}
	return postcode.ReplaceAllString(tidy, "${outer} ${inner}"), nil
}

func (c BinsClient) GetAddresses(postcode, token string) ([]Address, error) {
	canonical, err := canonicalize(postcode)
	if err != nil {
		return []Address{}, err
	}
	reqBody := []byte(`{
	    "type": "Query",
	    "aqs":
	    {
	        "properties":
	        {
	            "dodiCode": "designs_nlpgPremises",
	            "collectionCode": "Live",
	            "attributes":
	            [
	                "attributes_itemsTitle"
	            ]
	        },
	        "children":
	        [
	            {
	                "type": "Equals",
	                "properties":
	                {
	                    "__dataExplorerFilter": "attributes_premisesPostcode"
	                },
	                "children":
	                [
	                    {
	                        "type": "Attribute",
	                        "properties":
	                        {
	                            "attributeCode": "attributes_premisesPostcode",
	                            "value":
	                            []
	                        },
	                        "children":
	                        []
	                    },
	                    {
	                        "type": "String",
	                        "properties":
	                        {
	                            "attributeCode": "",
	                            "value":
	                            [
	                                "` + canonical + `"
	                            ]
	                        },
	                        "children":
	                        []
	                    }
	                ]
	            }
	        ]
	    }
	}`)
	target := c.ApiHost.JoinPath(queryUrl)
	query := target.Query()
	query.Set("pageSize", strconv.Itoa(100))
	query.Set("page", strconv.Itoa(1))
	target.RawQuery = query.Encode()

	cacheKey := target.JoinPath(canonical).String()
	if c.Cache != nil {
		if res, found := c.Cache.Get(cacheKey); found {
			return res.([]Address), nil
		}
	}

	req, err := http.NewRequest(http.MethodPost, target.String(), bytes.NewBuffer(reqBody))
	if err != nil {
		return []Address{}, err
	}
	req.Header.Add("Authorization", fmt.Sprint("Bearer ", token))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", userAgent)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return []Address{}, err
	}
	if resp.StatusCode != 200 {
		return []Address{}, fmt.Errorf("Status code %v fetching addresses for postcode", resp.StatusCode)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Address{}, err
	}

	type result struct {
		Results []struct {
			ItemId     string `json:"itemId"`
			Attributes []struct {
				AttributeCode string `json:"attributeCode"`
				Value         string `json:"value"`
			} `json:"attributes"`
		} `json:"results"`
	}

	var data result
	json.Unmarshal(respBody, &data)

	var addresses []Address
	for _, res := range data.Results {
		var name string
		for _, attribute := range res.Attributes {
			if attribute.AttributeCode == "attributes_itemsTitle" {
				name = tidy(attribute.Value)
				continue
			}
		}
		if res.ItemId != "" && name != "" {
			address := Address{
				res.ItemId,
				name,
			}
			addresses = append(addresses, address)
		}
	}

	slices.SortFunc(addresses, func(a, b Address) int {
		if natsort.Compare(a.Name, b.Name) {
			return -1
		}
		return 1
	})

	if c.Cache != nil {
		c.Cache.Add(cacheKey, addresses)
	}
	return addresses, nil
}
