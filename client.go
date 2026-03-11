package canalbox

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

const DefaultBaseURL = "https://grpvivendiafrica.my.site.com"

type Config struct {
	BaseURL     string
	Username    string
	Password    string
	SID         string
	AuraToken   string
	BrowserID   string
	OrgID       string
	Context     string
	PageURI     string
	PageScopeID string
	RenderCtx   string
}

type Client struct {
	cfg                Config
	client             *http.Client
	authMu             sync.Mutex
	auraMu             sync.Mutex
	auraCache          map[string]*AuraMetadata
	auraRequestCounter int64
	auraActionCounter  int64
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
		auraCache: make(map[string]*AuraMetadata),
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *Client) Config() *Config {
	return &c.cfg
}
