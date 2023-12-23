package client_test

import (
	"net/http"
)

const PropertyId = "property"
const Token = "token"

type fakeRoundTripper struct {
	Fn func(*http.Request) (*http.Response, error)
}

func (frt fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return frt.Fn(req)
}
