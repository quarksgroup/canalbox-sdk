package canalbox

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const DefaultBaseURL = "https://grpvivendiafrica.my.site.com"

type Config struct {
	BaseURL   string
	SID       string
	AuraToken string
	BrowserID string
	OrgID     string
	Context   string
	PageURI   string
}

type Client struct {
	cfg    Config
	client *http.Client
}

func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}

	jar, _ := cookiejar.New(nil)

	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{},
				ForceAttemptHTTP2: false,
			},
		},
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *Client) Config() *Config {
	return &c.cfg
}
