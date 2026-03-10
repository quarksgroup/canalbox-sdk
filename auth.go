package canalbox

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

func Login(baseURL, username, password string) (*Client, error) {
	cfg := Config{BaseURL: baseURL}
	jar, _ := cookiejar.New(nil)

	client := &Client{
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

	loginURL := baseURL + "/PortailDistributeur/login"

	req, err := http.NewRequest(http.MethodGet, loginURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create login: %w", err)
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login GET request: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	form := url.Values{}
	form.Add("un", username)
	form.Add("pw", password)
	form.Add("username", username)

	req, err = http.NewRequest(http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login POST request: %w", err)
	}
	defer resp.Body.Close()

	baseURLParsed, _ := url.Parse(baseURL)
	for _, c := range client.client.Jar.Cookies(baseURLParsed) {
		if c.Name == "sid" {
			client.cfg.SID = c.Value
		}
		if c.Name == "BrowserId" {
			client.cfg.BrowserID = c.Value
		}
		if c.Name == "oid" {
			client.cfg.OrgID = c.Value
		}
		if strings.HasPrefix(c.Name, "__Host-ERIC_PROD") {
			client.cfg.AuraToken = c.Value
		}
	}

	if client.cfg.SID == "" {
		return nil, fmt.Errorf("login failed: no SID cookie received")
	}

	subURL := baseURL + "/PortailDistributeur/s/subscription/Zuora__Subscription__c/Default?tabset-c5778=2"
	req, err = http.NewRequest(http.MethodGet, subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create subscription request: %w", err)
	}
	req.Header.Set("Referer", baseURL+"/PortailDistributeur/s/")

	resp, err = client.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("subscription page request: %w", err)
	}
	defer resp.Body.Close()

	for _, c := range client.client.Jar.Cookies(req.URL) {
		if strings.HasPrefix(c.Name, "__Host-ERIC_PROD") {
			client.cfg.AuraToken = c.Value
		}
	}

	if client.cfg.AuraToken == "" {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("login succeeded but could not extract aura.token")
	}

	client.cfg.Context = `{"mode":"PROD","fwuid":"YkVKdlZEd2t6eFplVFJNMGN2eVd5UTJEa1N5enhOU3R5QWl2VzNveFZTbGcxMy4tMjE0NzQ4MzY0OC45OTYxNDcy","app":"siteforce:communityApp","loaded":{"APPLICATION@markup://siteforce:communityApp":"1529_lI95rFcxq-le9BLgryC1ew","COMPONENT@markup://forceCommunity:reportChart":"805_A6VaKBntJ5kSj8Xst8h2Mg","COMPONENT@markup://forceCommunity:dashboard":"2174_6iY_-CT5MV4ib9ia3hfuHw","COMPONENT@markup://forceCommunity:objectHome":"2118_6I9SpwBIGLa8ytSlYL6Ujg","COMPONENT@markup://force:inputField":"1404_kTidaTm0Cp3r6gDe1UEI-A"},"dn":[],"globals":{},"uad":true}`

	return client, nil
}
