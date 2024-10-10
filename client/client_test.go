package client_test

import (
	"net/http"
)

const PropertyId = "property"
const BinId = "bin"
const WorkflowId = "workflow"
const Postcode = "E8 1EA"

type fakeRoundTripper struct {
	Fn func(*http.Request) (*http.Response, error)
}

func (frt fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return frt.Fn(req)
}
