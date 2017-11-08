package gontentful

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	timeout  = 10 * time.Second
	hostname = "cdn.contentful.com"

	pathSpace   = "/spaces/%s"
	pathEntries = pathSpace + "/entries"
)

type Client struct {
	client       *http.Client
	Options      *ClientOptions
	AfterRequest func(c *Client, req *http.Request, res *http.Response, elapsed time.Duration)
}

type ClientOptions struct {
	ApiToken string
	SpaceID  string
	ApiHost  string
}

func NewClient(options *ClientOptions) *Client {
	return &Client{
		Options: options,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) GetSpace(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathSpace, c.Options.SpaceID)
	return c.get(path, query)
}

func (c *Client) GetEntries(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, c.Options.SpaceID)
	return c.get(path, query)
}

func (c *Client) get(path string, query url.Values) ([]byte, error) {
	return c.req(http.MethodGet, path, query, nil)
}

func (c *Client) req(method string, path string, query url.Values, body io.Reader) ([]byte, error) {
	host := hostname
	if c.Options.ApiHost != "" {
		host = c.Options.ApiHost
	}

	u := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path,
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Options.ApiToken))

	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	start := time.Now()
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if c.AfterRequest != nil {
		c.AfterRequest(c, req, res, time.Since(start))
	}

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		return ioutil.ReadAll(res.Body)
	}

	apiError := c.parseError(req, res)

	// return apiError if it is not rate limit error
	if _, ok := apiError.(RateLimitExceededError); !ok {
		return nil, apiError
	}

	resetHeader := res.Header.Get("x-contentful-ratelimit-reset")

	// return apiError if Ratelimit-Reset header is not presented
	if resetHeader == "" {
		return nil, apiError
	}

	// wait X-Contentful-Ratelimit-Reset amount of seconds
	waitSeconds, err := strconv.Atoi(resetHeader)
	if err != nil {
		return nil, apiError
	}

	time.Sleep(time.Second * time.Duration(waitSeconds))

	return c.do(req)
}

func (c *Client) parseError(req *http.Request, res *http.Response) error {
	var e ErrorResponse
	defer res.Body.Close()
	err := json.NewDecoder(res.Body).Decode(&e)
	if err != nil {
		return err
	}

	apiError := APIError{
		req: req,
		res: res,
		err: &e,
	}

	switch errType := e.Sys.ID; errType {
	case "NotFound":
		return NotFoundError{apiError}
	case "RateLimitExceeded":
		return RateLimitExceededError{apiError}
	case "AccessTokenInvalid":
		return AccessTokenInvalidError{apiError}
	case "ValidationFailed":
		return ValidationFailedError{apiError}
	case "VersionMismatch":
		return VersionMismatchError{apiError}
	case "Conflict":
		return VersionMismatchError{apiError}
	default:
		return e
	}
}
