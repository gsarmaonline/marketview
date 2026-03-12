package nse

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
)

const (
	baseURL   = "https://www.nseindia.com"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
)

// Client is an HTTP client that handles NSE's cookie-based session requirement.
type Client struct {
	http *http.Client
}

// New creates a Client and establishes a session with NSE by hitting the homepage.
func New() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	c := &Client{
		http: &http.Client{Jar: jar},
	}

	if err := c.initSession(); err != nil {
		return nil, fmt.Errorf("initialising NSE session: %w", err)
	}

	return c, nil
}

// initSession hits the NSE homepage to obtain session cookies.
func (c *Client) initSession() error {
	req, err := http.NewRequest(http.MethodGet, baseURL, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// Get performs a GET request to the given NSE API URL with proper headers.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	return c.http.Do(req)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", baseURL+"/")
}
