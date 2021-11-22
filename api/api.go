// Package api provides API wrappers for some of Lovense's API.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultForm are the default form values.
var DefaultForm = url.Values{
	"appVersion": {"5.1.6"},
	"version":    {"2"},
	"platform":   {"android"},
}

// DefaultHeader are the default request headers.
var DefaultHeader = http.Header{
	"User-Agent": {"okhttp/3.12.3"},
}

// RequestOpt is the type for an API option.
type RequestOpt func(*Client, *http.Request)

// WithPOSTForm injects the given form as an x-www-form-urlencoded body.
func WithPOSTForm(form url.Values) RequestOpt {
	return func(c *Client, r *http.Request) {
		newForm := make(url.Values, len(form)+len(c.DefaultForm))
		for k, v := range c.DefaultForm {
			newForm[k] = v
		}
		for k, v := range form {
			newForm[k] = append(newForm[k], v...)
		}

		encoded := newForm.Encode()

		r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		r.Header.Set("Content-Length", strconv.Itoa(len(encoded)))
		r.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(encoded)), nil
		}
		r.Body, _ = r.GetBody()
	}
}

// WithHeader injects the given header.
func WithHeader(h http.Header) RequestOpt {
	return func(c *Client, r *http.Request) {
		for k, v := range h {
			r.Header[k] = v
		}
	}
}

// Client is a general API client.
type Client struct {
	*http.Client
	Host          string // apps.lovense.com
	DefaultForm   url.Values
	DefaultHeader http.Header
}

// NewClient returns a new client.
func NewClient() *Client {
	client := *http.DefaultClient
	client.Timeout = time.Minute

	return &Client{
		Client:      &client,
		Host:        "apps.lovense.com",
		DefaultForm: DefaultForm,
	}
}

// DoGET sends a GET to the given URL.
func (c *Client) DoGET(path string, outJSON interface{}, opts ...RequestOpt) error {
	return c.DoJSON("GET", path, outJSON, opts...)
}

// DoPOST sends a POST to the given URL. If outJSON is not nil, then a JSON body
// is read.
func (c *Client) DoPOST(path string, outJSON interface{}, opts ...RequestOpt) error {
	return c.DoJSON("POST", path, outJSON, opts...)
}

// DoJSON sends a HTTP request and unmarshals into the given outJSON.
func (c *Client) DoJSON(method, path string, outJSON interface{}, opts ...RequestOpt) error {
	r, err := c.Do(method, path, opts...)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		serverErr := ServerError{Status: r.StatusCode}
		json.NewDecoder(r.Body).Decode(&serverErr) // error doesn't matter
		return &serverErr
	}

	if outJSON != nil {
		if err := json.NewDecoder(r.Body).Decode(outJSON); err != nil {
			return fmt.Errorf("cannot decode JSON response: %w", err)
		}
	}

	return nil
}

// Do sends a HTTP request and returns a typical HTTP response.
func (c *Client) Do(method, path string, opts ...RequestOpt) (*http.Response, error) {
	fullURL := path

	// awful hack
	if !strings.Contains(path, "://") {
		u := url.URL{
			Scheme: "https",
			Host:   c.Host,
			Path:   path,
		}
		fullURL = u.String()
	}

	// TODO: string + reparse is dumb
	r, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}

	for k, v := range c.DefaultHeader {
		r.Header[k] = v
	}

	for _, opt := range opts {
		opt(c, r)
	}

	return c.Client.Do(r)
}

// ServerError is the server error. It implements error.
type ServerError struct {
	ResponseBody
	Status int
}

// Error implements error.
func (e *ServerError) Error() string {
	if e.Code == 0 {
		return fmt.Sprintf(
			"server returned status code %d",
			e.Status,
		)
	}

	return fmt.Sprintf(
		"server returned status code %d, server code %d %q",
		e.Status, e.Code, e.Message,
	)
}

// ResponseBody is the general response body that the backend responds with.
type ResponseBody struct {
	Code    int64       `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Result  bool        `json:"result"`
}

// RawData returns Data as a raw JSON message.
func (body ResponseBody) RawData() json.RawMessage {
	b, _ := json.Marshal(body.Data)
	return b
}
