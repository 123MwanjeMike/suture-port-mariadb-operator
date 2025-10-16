package http

import (
	"net/http"
	"os"

	"k8s.io/client-go/rest"
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

// WrapRestConfigWithSutureID wraps a Kubernetes rest.Config to add the Suture_ID header to all requests
func WrapRestConfigWithSutureID(config *rest.Config) {
	if config == nil {
		return
	}
	
	// Set the WrapTransport function to add the Suture_ID header
	originalWrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		// If there's an existing wrapper, apply it first
		if originalWrap != nil {
			rt = originalWrap(rt)
		}
		// Then wrap with our Suture_ID transport
		return NewHeadersTransport(rt, map[string]string{})
	}
}
