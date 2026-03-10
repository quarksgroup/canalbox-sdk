package canalbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Subscription struct {
	Name                      string          `json:"Name"`
	Id                        string          `json:"Id"`
	BoxNumber__c              string          `json:"BoxNumber__c"`
	ExpectedRenewalDate__c    string          `json:"ExpectedRenewalDate__c"`
	Zuora__Account__c         string          `json:"Zuora__Account__c"`
	Zuora__CustomerAccount__c string          `json:"Zuora__CustomerAccount__c"`
	SUB_T_Produit_De_Base__c  string          `json:"SUB_T_Produit_De_Base__c"`
	Zuora__Account__r         Account         `json:"Zuora__Account__r"`
	Zuora__CustomerAccount__r CustomerAccount `json:"Zuora__CustomerAccount__r"`
}

type Account struct {
	Phone            string `json:"Phone"`
	ACC_Indicatif__c string `json:"ACC_Indicatif__c"`
	Name             string `json:"Name"`
	QU_Quartier__c   string `json:"QU_Quartier__c"`
	Id               string `json:"Id"`
}

type CustomerAccount struct {
	Zuora__AccountNumber__c string `json:"Zuora__AccountNumber__c"`
	Id                      string `json:"Id"`
}

type Action struct {
	Id          string      `json:"id"`
	Descriptor  string      `json:"descriptor"`
	ReturnValue ReturnValue `json:"returnValue"`
	Error       []APIError  `json:"error"`
	State       string      `json:"state"`
}

type ReturnValue struct {
	ReturnValue any  `json:"returnValue"`
	Cacheable   bool `json:"cacheable"`
}

type APIError struct {
	Message string `json:"message"`
}

type Response struct {
	Actions []Action `json:"actions"`
}

func (c *Client) buildRequest(searchKey string) (*http.Request, error) {
	url := fmt.Sprintf("%s/PortailDistributeur/s/sfsites/aura?r=32&aura.ApexAction.execute=1", c.cfg.BaseURL)

	payload := fmt.Sprintf(`message=%%7B%%22actions%%22%%3A%%5B%%7B%%22id%%22%%3A%%22825%%3Ba%%22%%2C%%22descriptor%%22%%3A%%22aura%%3A%%2F%%2FApexActionController%%2FACTION%%24execute%%22%%2C%%22callingDescriptor%%22%%3A%%22UNKNOWN%%22%%2C%%22params%%22%%3A%%7B%%22namespace%%22%%3A%%22%%22%%2C%%22classname%%22%%3A%%22AP02_DistributorAccount%%22%%2C%%22method%%22%%3A%%22getListSubscription%%22%%2C%%22params%%22%%3A%%7B%%22searchKey%%22%%3A%%22%s%%22%%7D%%2C%%22cacheable%%22%%3Afalse%%2C%%22isContinuation%%22%%3Afalse%%7D%%7D%%5D%%7D&aura.context=%s&aura.pageURI=%s&aura.token=%s`,
		searchKey,
		urlEncode(c.cfg.Context),
		urlEncode(c.cfg.PageURI),
		c.cfg.AuraToken,
	)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("construct request: %w", err)
	}

	rawCookies := fmt.Sprintf("BrowserId=%s; oid=%s; sid=%s",
		c.cfg.BrowserID,
		c.cfg.OrgID,
		c.cfg.SID,
	)
	req.Header.Set("Cookie", rawCookies)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", c.cfg.BaseURL)
	req.Header.Set("Referer", c.cfg.BaseURL+"/PortailDistributeur/s/subscription/Zuora__Subscription__c/Default?tabset-c5778=2")

	return req, nil
}

func (c *Client) GetSubscription(searchKey string) ([]Subscription, error) {
	req, err := c.buildRequest(searchKey)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var response Response
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(bodyBytes))
	}

	if len(response.Actions) == 0 {
		return nil, fmt.Errorf("no actions in response")
	}

	action := response.Actions[0]
	if action.State != "SUCCESS" {
		if len(action.Error) > 0 {
			return nil, fmt.Errorf("API error: %s", action.Error[0].Message)
		}
		return nil, fmt.Errorf("API returned error state: %s", action.State)
	}

	var rawResp map[string]any
	if err := json.Unmarshal(bodyBytes, &rawResp); err != nil {
		return nil, fmt.Errorf("parse raw response: %w", err)
	}

	actions := rawResp["actions"].([]any)
	firstAction := actions[0].(map[string]any)
	returnValue := firstAction["returnValue"].(map[string]any)
	returnValueMap := returnValue["returnValue"].([]any)

	var subs []Subscription
	for _, item := range returnValueMap {
		m := item.(map[string]any)
		sub := Subscription{
			Name:                      toString(m["Name"]),
			Id:                        toString(m["Id"]),
			BoxNumber__c:              toString(m["BoxNumber__c"]),
			ExpectedRenewalDate__c:    toString(m["ExpectedRenewalDate__c"]),
			Zuora__Account__c:         toString(m["Zuora__Account__c"]),
			Zuora__CustomerAccount__c: toString(m["Zuora__CustomerAccount__c"]),
			SUB_T_Produit_De_Base__c:  toString(m["SUB_T_Produit_De_Base__c"]),
		}

		if accountR, ok := m["Zuora__Account__r"].(map[string]any); ok {
			sub.Zuora__Account__r = Account{
				Phone:            toString(accountR["Phone"]),
				ACC_Indicatif__c: toString(accountR["ACC_Indicatif__c"]),
				Name:             toString(accountR["Name"]),
				QU_Quartier__c:   toString(accountR["QU_Quartier__c"]),
				Id:               toString(accountR["Id"]),
			}
		}

		if custAccountR, ok := m["Zuora__CustomerAccount__r"].(map[string]any); ok {
			sub.Zuora__CustomerAccount__r = CustomerAccount{
				Zuora__AccountNumber__c: toString(custAccountR["Zuora__AccountNumber__c"]),
				Id:                      toString(custAccountR["Id"]),
			}
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func urlEncode(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, " ", "%20")
	s = strings.ReplaceAll(s, "!", "%21")
	s = strings.ReplaceAll(s, "#", "%23")
	s = strings.ReplaceAll(s, "$", "%24")
	s = strings.ReplaceAll(s, "&", "%26")
	s = strings.ReplaceAll(s, "=", "%3D")
	s = strings.ReplaceAll(s, "+", "%2B")
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, "/", "%2F")
	s = strings.ReplaceAll(s, "?", "%3F")
	s = strings.ReplaceAll(s, "@", "%40")
	s = strings.ReplaceAll(s, "[", "%5B")
	s = strings.ReplaceAll(s, "]", "%5D")
	return s
}
