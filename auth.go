package canalbox

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func Login(baseURL, username, password string) (*Client, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	client := NewClient(Config{BaseURL: baseURL})

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

	client.updateCookies(resp.Request.URL)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read subscription page: %w", err)
	}

	if _, err := client.cacheAuraMetadata(defaultSubscriptionListPageURI, resp.Request.URL, body); err != nil {
		return nil, err
	}

	return client, nil
}
