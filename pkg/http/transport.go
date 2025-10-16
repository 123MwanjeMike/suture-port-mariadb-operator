package http

import (
	"net/http"
	"os"
)

type HeadersTransport struct {
	roundTripper http.RoundTripper
	headers      map[string]string
}

func NewHeadersTransport(rt http.RoundTripper, headers map[string]string) http.RoundTripper {
	transport := &HeadersTransport{
		roundTripper: rt,
		headers:      headers,
	}
	if transport.roundTripper == nil {
		transport.roundTripper = http.DefaultTransport
	}
	return transport
}

func (t *HeadersTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Suture_ID", os.Getenv("SUTURE_ID"))
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
	}
	return t.roundTripper.RoundTrip(req)
}
