// Package httpclient is the HTTP client shared by the devices and the external APIs.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client wraps one http.Client, so connections are reused. Timeouts come from the context.
type Client struct {
	client *http.Client
}

func New() *Client {
	return &Client{client: &http.Client{}}
}

func (c *Client) Get(ctx context.Context, url string) ([]byte, int, error) {
	return c.Do(ctx, http.MethodGet, url, "", nil)
}

func (c *Client) Post(ctx context.Context, url string, body string, headers map[string]string) ([]byte, int, error) {
	return c.Do(ctx, http.MethodPost, url, body, headers)
}

// Do sends the request and returns the response body and status. A network failure or a status
// outside 2xx is an error; the body and status are still returned when available.
func (c *Client) Do(ctx context.Context, method string, url string, body string, headers map[string]string) ([]byte, int, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response.StatusCode, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return responseBody, response.StatusCode, fmt.Errorf("%s %s: status %d", method, url, response.StatusCode)
	}
	return responseBody, response.StatusCode, nil
}
