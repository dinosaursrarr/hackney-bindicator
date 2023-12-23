package client

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/anaskhan96/soup"
)

func (c BinsClient) GetAccessToken() (string, error) {
    req, err := http.NewRequest(http.MethodGet, c.StartUrl.String(), nil)
    if err != nil {
        return "", err
    }
    resp, err := c.HttpClient.Do(req)
    if err != nil {
        return "", err
    }
    if resp.StatusCode != 200 {
        return "", fmt.Errorf("Status code %v fetching access token", resp.StatusCode)
    }
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    doc := soup.HTMLParse(string(body))
    scripts := doc.FindAll("script")
    for _, script := range scripts {
        t := script.Text()
        prefix := strings.Index(script.Text(), "ALLOY_APP_TOKEN")
        if prefix == -1 {
            continue
        }
        start := strings.Index(t[prefix:], "\"") + prefix + 1
        end := strings.Index(t[start:], "\"") + start
        return t[start:end], nil
    }

    return "", errors.New("Could not find access token in response")
}