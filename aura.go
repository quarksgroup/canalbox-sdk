package canalbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const defaultAuraContext = `{"mode":"PROD","fwuid":"YkVKdlZEd2t6eFplVFJNMGN2eVd5UTJEa1N5enhOU3R5QWl2VzNveFZTbGcxMy4tMjE0NzQ4MzY0OC45OTYxNDcy","app":"siteforce:communityApp","loaded":{"APPLICATION@markup://siteforce:communityApp":"1529_lI95rFcxq-le9BLgryC1ew","COMPONENT@markup://forceCommunity:reportChart":"805_A6VaKBntJ5kSj8Xst8h2Mg","COMPONENT@markup://forceCommunity:dashboard":"2174_6iY_-CT5MV4ib9ia3hfuHw","COMPONENT@markup://forceCommunity:objectHome":"2118_6I9SpwBIGLa8ytSlYL6Ujg","COMPONENT@markup://force:inputField":"1404_kTidaTm0Cp3r6gDe1UEI-A"},"dn":[],"globals":{},"uad":true}`

var (
	auraContextRegex = regexp.MustCompile(`(?s)aura\.context\s*=\s*['"](\{.*?\})['"]`)
	auraPageURIRegex = regexp.MustCompile(`(?s)aura\.pageURI\s*=\s*['"]([^'"\s]+)['"]`)
	pageScopeRegex   = regexp.MustCompile(`(?i)pageScopeId\s*[:=]\s*['"]([^'"\s]+)['"]`)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AuraMetadata struct {
	Context     string
	PageURI     string
	PageScopeID string
	RenderCtx   string
}

func (c *Client) ensureAuraMetadata(pageURI string) (*AuraMetadata, error) {
	normalized := normalizePageURI(pageURI)
	c.auraMu.Lock()
	meta, ok := c.auraCache[normalized]
	c.auraMu.Unlock()
	if ok && meta.Context != "" {
		return meta, nil
	}

	fullURL := normalized
	if !strings.HasPrefix(normalized, "http://") && !strings.HasPrefix(normalized, "https://") {
		fullURL = c.cfg.BaseURL + normalized
	}
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("construct aura context request: %w", err)
	}
	req.Header.Set("Referer", c.cfg.BaseURL+"/PortailDistributeur/s/")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch aura page %s: %w", normalized, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read aura page: %w", err)
	}

	c.updateCookies(resp.Request.URL)

	meta, err = c.cacheAuraMetadata(normalized, resp.Request.URL, body)
	if err != nil {
		return nil, err
	}

	c.auraMu.Lock()
	c.auraCache[normalized] = meta
	c.auraMu.Unlock()

	return meta, nil
}

func (c *Client) cacheAuraMetadata(pageURI string, u *url.URL, body []byte) (*AuraMetadata, error) {
	ctxValue := extractAuraContext(body)
	if ctxValue == "" {
		if c.cfg.Context != "" {
			ctxValue = c.cfg.Context
		} else {
			ctxValue = defaultAuraContext
		}
	}

	pageURIValue := extractAuraPageURI(body)
	if pageURIValue == "" {
		pageURIValue = pageURI
	}

	pageScopeID := extractPageScopeID(body)
	if pageScopeID == "" {
		pageScopeID = c.cfg.PageScopeID
	}
	if pageScopeID == "" {
		pageScopeID = randomUUIDLike()
	}

	meta := &AuraMetadata{
		Context:     ctxValue,
		PageURI:     pageURIValue,
		PageScopeID: pageScopeID,
		RenderCtx:   c.cookieValue(u, "renderCtx"),
	}

	c.cfg.Context = ctxValue
	c.cfg.PageURI = pageURIValue
	c.cfg.PageScopeID = pageScopeID
	if meta.RenderCtx != "" {
		c.cfg.RenderCtx = meta.RenderCtx
	}

	return meta, nil
}

func (c *Client) updateCookies(u *url.URL) {
	for _, ck := range c.client.Jar.Cookies(u) {
		switch ck.Name {
		case "sid":
			c.cfg.SID = ck.Value
		case "BrowserId":
			c.cfg.BrowserID = ck.Value
		case "oid":
			c.cfg.OrgID = ck.Value
		case "__Host-ERIC_PROD":
			c.cfg.AuraToken = ck.Value
		default:
			if strings.HasPrefix(ck.Name, "__Host-ERIC_PROD") {
				c.cfg.AuraToken = ck.Value
			}
		}
	}
}

func (c *Client) cookieValue(u *url.URL, name string) string {
	for _, ck := range c.client.Jar.Cookies(u) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func normalizePageURI(uri string) string {
	if uri == "" {
		return "/"
	}
	if strings.HasPrefix(uri, "http") {
		return uri
	}
	if strings.HasPrefix(uri, "/") {
		return uri
	}
	return "/" + uri
}

func extractAuraContext(body []byte) string {
	matches := auraContextRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(string(matches[1]))
}

func extractAuraPageURI(body []byte) string {
	matches := auraPageURIRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(string(matches[1]))
}

func extractPageScopeID(body []byte) string {
	matches := pageScopeRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(string(matches[1]))
}

func (c *Client) doAuraAction(ctx context.Context, pageURI, className, method string, params any) (*Response, error) {
	action := map[string]any{
		"id":                fmt.Sprintf("%d;a", atomic.AddInt64(&c.auraActionCounter, 1)),
		"descriptor":        "aura://ApexActionController/ACTION$execute",
		"callingDescriptor": "UNKNOWN",
		"params": map[string]any{
			"namespace":      "",
			"classname":      className,
			"method":         method,
			"params":         params,
			"cacheable":      false,
			"isContinuation": false,
		},
	}

	ldsEndpoint := fmt.Sprintf("ApexActionController.execute:%s.%s", className, method)
	return c.doAuraRequest(ctx, pageURI, "aura.ApexAction.execute=1", action, ldsEndpoint)
}

func (c *Client) doAuraRequest(ctx context.Context, pageURI, queryParam string, action map[string]any, ldsEndpoint string) (*Response, error) {
	ctx = ensureContext(ctx)
	meta, err := c.ensureAuraMetadata(pageURI)
	if err != nil {
		return nil, err
	}

	message := map[string]any{"actions": []map[string]any{action}}
	data, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal aura message: %w", err)
	}

	form := url.Values{}
	form.Set("message", string(data))
	form.Set("aura.context", meta.Context)
	form.Set("aura.pageURI", meta.PageURI)
	form.Set("aura.token", c.cfg.AuraToken)

	endpoint := fmt.Sprintf("%s/PortailDistributeur/s/sfsites/aura?r=%d&%s", c.cfg.BaseURL, atomic.AddInt64(&c.auraRequestCounter, 1), queryParam)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create aura POST: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", c.cfg.BaseURL)
	req.Header.Set("Referer", c.cfg.BaseURL+meta.PageURI)
	req.Header.Set("priority", "u=1, i")
	if ldsEndpoint != "" {
		req.Header.Set("x-sfdc-lds-endpoints", ldsEndpoint)
	}
	req.Header.Set("x-sfdc-page-scope-id", meta.PageScopeID)
	req.Header.Set("x-sfdc-request-id", fmt.Sprintf("%016x", time.Now().UnixNano()))
	req.Header.Set("dnt", "1")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("x-b3-sampled", "0")
	req.Header.Set("x-b3-spanid", randomHex(16))
	req.Header.Set("x-b3-traceid", randomHex(16))
	req.Header.Set("User-Agent", "CanalBoxSDK/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute aura action: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read aura response: %w", err)
	}
	c.updateCookies(resp.Request.URL)
	bodyText := string(body)
	if resp.StatusCode != http.StatusOK {
		if isSessionExpiredStatus(resp.StatusCode) || isSessionExpiredMessage(bodyText) {
			return nil, wrapSessionExpired(fmt.Sprintf("aura response status %d", resp.StatusCode))
		}

		return nil, fmt.Errorf("aura response status %d: %s", resp.StatusCode, string(body))
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		if isSessionExpiredMessage(bodyText) || strings.Contains(strings.ToLower(bodyText), "/portaildistributeur/login") {
			return nil, wrapSessionExpired("redirected to login")
		}

		return nil, fmt.Errorf("parse aura response: %w", err)
	}

	return &response, nil
}

func randomHex(length int) string {
	const letters = "0123456789abcdef"
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = letters[rand.Intn(len(letters))]
	}
	return string(buf)
}

func randomUUIDLike() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", randomHex(8), randomHex(4), randomHex(4), randomHex(4), randomHex(12))
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
