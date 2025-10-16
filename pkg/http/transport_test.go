package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

func TestHeadersTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		sutureID       string
		customHeaders  map[string]string
		expectSutureID string
	}{
		{
			name:           "with SUTURE_ID set",
			sutureID:       "test-suture-id-123",
			customHeaders:  map[string]string{"X-Custom": "custom-value"},
			expectSutureID: "test-suture-id-123",
		},
		{
			name:           "without SUTURE_ID set",
			sutureID:       "",
			customHeaders:  map[string]string{},
			expectSutureID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variable
			if tt.sutureID != "" {
				os.Setenv("SUTURE_ID", tt.sutureID)
				defer os.Unsetenv("SUTURE_ID")
			} else {
				os.Unsetenv("SUTURE_ID")
			}

			// Create a test server to capture the request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify headers
				assert.Equal(t, tt.expectSutureID, r.Header.Get("Suture_ID"))
				for k, v := range tt.customHeaders {
					assert.Equal(t, v, r.Header.Get(k))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create the transport
			transport := NewHeadersTransport(http.DefaultTransport, tt.customHeaders)

			// Create a request
			req, err := http.NewRequest("GET", server.URL, nil)
			assert.NoError(t, err)

			// Execute the request
			resp, err := transport.RoundTrip(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})
	}
}

func TestHeadersTransport_RoundTrip_WithBody(t *testing.T) {
	sutureID := "test-suture-id-with-body"
	os.Setenv("SUTURE_ID", sutureID)
	defer os.Unsetenv("SUTURE_ID")

	// Create a test server to capture the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, sutureID, r.Header.Get("Suture_ID"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create the transport
	transport := NewHeadersTransport(http.DefaultTransport, map[string]string{})

	// Create a request with a body
	body := `{"test": "data"}`
	req, err := http.NewRequest("POST", server.URL, io.NopCloser(io.Reader(&mockReader{data: body})))
	assert.NoError(t, err)

	// Execute the request
	resp, err := transport.RoundTrip(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// mockReader is a simple mock reader for testing
type mockReader struct {
	data string
	pos  int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func TestWrapRestConfigWithSutureID(t *testing.T) {
	sutureID := "test-config-suture-id"
	os.Setenv("SUTURE_ID", sutureID)
	defer os.Unsetenv("SUTURE_ID")

	tests := []struct {
		name   string
		config *rest.Config
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name: "valid config without existing WrapTransport",
			config: &rest.Config{
				Host: "https://example.com",
			},
		},
		{
			name: "valid config with existing WrapTransport",
			config: &rest.Config{
				Host: "https://example.com",
				WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
					// Existing wrapper
					return rt
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			WrapRestConfigWithSutureID(tt.config)

			if tt.config == nil {
				return // Nothing to verify for nil config
			}

			// Verify that WrapTransport is set
			assert.NotNil(t, tt.config.WrapTransport)

			// Test that the WrapTransport actually wraps the transport correctly
			testTransport := &mockRoundTripper{}
			wrappedTransport := tt.config.WrapTransport(testTransport)
			assert.NotNil(t, wrappedTransport)

			// Verify that the wrapped transport adds Suture_ID header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, sutureID, r.Header.Get("Suture_ID"))
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			req, err := http.NewRequest("GET", server.URL, nil)
			assert.NoError(t, err)

			resp, err := wrappedTransport.RoundTrip(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			resp.Body.Close()
		})
	}
}

// mockRoundTripper is a simple mock for testing
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return http.DefaultTransport.RoundTrip(req)
}
