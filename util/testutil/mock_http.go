package testutil

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

// MockResponse defines a stubbed HTTP response status and body.
type MockResponse struct {
	StatusCode int
	Body       string
}

// MockRoundTripper implements http.RoundTripper for clean declarative test mocking.
type MockRoundTripper struct {
	Stubs map[string]MockResponse
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	urlStr := req.URL.String()
	for pattern, stub := range m.Stubs {
		if strings.Contains(urlStr, pattern) {
			return &http.Response{
				StatusCode: stub.StatusCode,
				Body:       io.NopCloser(bytes.NewBufferString(stub.Body)),
				Header:     make(http.Header),
			}, nil
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
		Header:     make(http.Header),
	}, nil
}

// NewMockHTTPClient creates an *http.Client configured with declarative URL stubs.
func NewMockHTTPClient(stubs map[string]MockResponse) *http.Client {
	return &http.Client{
		Transport: &MockRoundTripper{Stubs: stubs},
	}
}
