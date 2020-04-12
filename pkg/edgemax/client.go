// Package edgemax implements a client for Ubiquiti EdgeMAX devices.
package edgemax

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	// userAgent is the default user agent this package will report to the
	// EdgeMAX device.
	userAgent = "github.com/mdlayher/edgemax"
)

// InsecureHTTPClient creates a *http.Client which does not verify an EdgeMAX
// device's certificate chain and hostname.
//
// Please think carefully before using this client: it should only be used
// with self-hosted, internal EdgeMAX devices.
func InsecureHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// A Client is a client for a Ubiquiti EdgeMAX device.
//
// Client.Login must be called and return a nil error before any additional
// actions can be performed with a Client.
type Client struct {
	UserAgent string

	apiURL *url.URL
	client *http.Client
}

// NewClient creates a new Client, using the input EdgeMAX device address
// and an optional HTTP client.  If no HTTP client is specified, a default
// one will be used.
//
// If working with a self-hosted EdgeMAX device which does not have a valid
// TLS certificate, InsecureHTTPClient can be used.
//
// Client.Login must be called and return a nil error before any additional
// actions can be performed with a Client.
func NewClient(addr string, client *http.Client) (*Client, error) {
	// Trim trailing slash to ensure sane path creation in other methods
	u, err := url.Parse(strings.TrimRight(addr, "/"))
	if err != nil {
		return nil, err
	}

	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	if client.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		client.Jar = jar
	}

	c := &Client{
		UserAgent: userAgent,

		apiURL: u,
		client: client,
	}

	return c, nil
}

// Login authenticates against the EdgeMAX device using the specified username
// and password.  Login must be called and return a nil error before any
// additional actions can be performed.
func (c *Client) Login(username string, password string) error {
	v := make(url.Values, 2)
	v.Set("username", username)
	v.Set("password", password)

	_, err := c.client.PostForm(c.apiURL.String(), v)
	return err
}

// newRequest creates a new HTTP request, using the specified HTTP method and
// API endpoint.
func (c *Client) newRequest(method string, endpoint string) (*http.Request, error) {
	rel, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u := c.apiURL.ResolveReference(rel)

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Needed to allow authentication to many HTTP endpoints
	req.Header.Set("Referer", c.apiURL.String())

	req.Header.Add("User-Agent", c.UserAgent)

	return req, nil
}

// do performs an HTTP request using req and unmarshals the result onto v, if
// v is not nil.
func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if v == nil {
		return res, nil
	}

	return res, json.NewDecoder(res.Body).Decode(v)
}
