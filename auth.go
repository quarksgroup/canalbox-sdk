package canalbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultBaseURL = "https://grpvivendiafrica.my.site.com"
)

type Options struct {
	BaseURL *string
}

func Login(ctx context.Context, username, password string, opts *Options) (*Client, error) {
	baseURL := defaultBaseURL
	if opts != nil && opts.BaseURL != nil {
		baseURL = *opts.BaseURL
	}

	client := NewClient(Config{BaseURL: baseURL})
	if err := client.login(ctx, username, password); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) Refresh(ctx context.Context) error {
	c.authMu.Lock()
	defer c.authMu.Unlock()

	if strings.TrimSpace(c.cfg.Username) == "" || strings.TrimSpace(c.cfg.Password) == "" {
		return fmt.Errorf("cannot refresh session: missing credentials")
	}

	c.auraMu.Lock()
	c.auraCache = make(map[string]*AuraMetadata)
	c.auraMu.Unlock()

	return c.login(ctx, c.cfg.Username, c.cfg.Password)
}

func (c *Client) login(ctx context.Context, username, password string) error {
	ctx = ensureContext(ctx)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	c.cfg.Username = username
	c.cfg.Password = password

	loginURL := c.cfg.BaseURL + "/PortailDistributeur/login"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loginURL, nil)
	if err != nil {
		return fmt.Errorf("create login: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("login GET request: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	form := url.Values{}
	form.Add("un", username)
	form.Add("pw", password)
	form.Add("username", username)

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = c.client.Do(req)
	if err != nil {
		return fmt.Errorf("login POST request: %w", err)
	}
	defer resp.Body.Close()

	baseURLParsed, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("parse base URL: %w", err)
	}
	for _, ck := range c.client.Jar.Cookies(baseURLParsed) {
		if ck.Name == "sid" {
			c.cfg.SID = ck.Value
		}
		if ck.Name == "BrowserId" {
			c.cfg.BrowserID = ck.Value
		}
		if ck.Name == "oid" {
			c.cfg.OrgID = ck.Value
		}
		if strings.HasPrefix(ck.Name, "__Host-ERIC_PROD") {
			c.cfg.AuraToken = ck.Value
		}
	}

	if c.cfg.SID == "" {
		return fmt.Errorf("login failed: no SID cookie received")
	}

	subURL := c.cfg.BaseURL + "/PortailDistributeur/s/subscription/Zuora__Subscription__c/Default?tabset-c5778=2"
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, subURL, nil)
	if err != nil {
		return fmt.Errorf("create subscription request: %w", err)
	}
	req.Header.Set("Referer", c.cfg.BaseURL+"/PortailDistributeur/s/")

	resp, err = c.client.Do(req)
	if err != nil {
		return fmt.Errorf("subscription page request: %w", err)
	}
	defer resp.Body.Close()

	c.updateCookies(resp.Request.URL)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read subscription page: %w", err)
	}

	meta, err := c.cacheAuraMetadata(defaultSubscriptionListPageURI, resp.Request.URL, body)
	if err != nil {
		return err
	}

	c.auraMu.Lock()
	c.auraCache[defaultSubscriptionListPageURI] = meta
	c.auraMu.Unlock()

	return nil
}
