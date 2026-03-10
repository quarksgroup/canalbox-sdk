package canalbox

import (
	"context"
	"fmt"
	"strings"
)

func (s Subscription) PageURI() (string, error) {
	return SubscriptionPageURI(s.Id, s.SubscriptionNumber)
}

func SubscriptionPageURI(subscriptionID, subscriptionNumber string) (string, error) {
	if subscriptionID == "" {
		return "", fmt.Errorf("subscription id is required")
	}

	slug := normalizeSubscriptionNumber(subscriptionNumber)
	if slug == "" {
		return "", fmt.Errorf("subscription number is required")
	}

	return fmt.Sprintf("/PortailDistributeur/s/subscription/%s/%s", subscriptionID, slug), nil
}

func (c *Client) GetSubscription(searchKey string) ([]Subscription, error) {
	return c.GetSubscriptionContext(context.Background(), searchKey)
}

func (c *Client) GetSubscriptionContext(ctx context.Context, searchKey string) ([]Subscription, error) {
	ctx = ensureContext(ctx)
	params := map[string]any{"searchKey": searchKey}

	response, err := c.doAuraAction(ctx, defaultSubscriptionListPageURI, "AP02_DistributorAccount", "getListSubscription", params)
	if err != nil {
		return nil, err
	}

	action, err := firstAction(response)
	if err != nil {
		return nil, err
	}

	var rawSubs []map[string]any
	if err := decodeReturnValue(action.ReturnValue.Value, &rawSubs); err != nil {
		return nil, fmt.Errorf("parse subscription list: %w", err)
	}

	subs := make([]Subscription, 0, len(rawSubs))

	for _, item := range rawSubs {
		subscriptionNumber := toString(item["SubscriptionNumber"])
		if subscriptionNumber == "" {
			subscriptionNumber = toString(item["Zuora__SubscriptionNumber__c"])
		}
		if subscriptionNumber == "" {
			subscriptionNumber = toString(item["Name"])
		}

		sub := Subscription{
			Name:                      toString(item["Name"]),
			Id:                        toString(item["Id"]),
			BoxNumber__c:              toString(item["BoxNumber__c"]),
			ExpectedRenewalDate__c:    toString(item["ExpectedRenewalDate__c"]),
			Zuora__Account__c:         toString(item["Zuora__Account__c"]),
			Zuora__CustomerAccount__c: toString(item["Zuora__CustomerAccount__c"]),
			SUB_T_Produit_De_Base__c:  toString(item["SUB_T_Produit_De_Base__c"]),
			SubscriptionNumber:        subscriptionNumber,
		}

		if account, ok := item["Zuora__Account__r"].(map[string]any); ok {
			sub.Zuora__Account__r = Account{
				Phone:            toString(account["Phone"]),
				ACC_Indicatif__c: toString(account["ACC_Indicatif__c"]),
				Name:             toString(account["Name"]),
				QU_Quartier__c:   toString(account["QU_Quartier__c"]),
				Id:               toString(account["Id"]),
			}
		}

		if cust, ok := item["Zuora__CustomerAccount__r"].(map[string]any); ok {
			sub.Zuora__CustomerAccount__r = CustomerAccount{
				Zuora__AccountNumber__c: toString(cust["Zuora__AccountNumber__c"]),
				Id:                      toString(cust["Id"]),
			}
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func (c *Client) GetSubscriptionByBoxNumber(boxNumber string) (*Subscription, error) {
	return c.GetSubscriptionByBoxNumberContext(context.Background(), boxNumber)
}

func (c *Client) GetSubscriptionByBoxNumberContext(ctx context.Context, boxNumber string) (*Subscription, error) {
	requested := normalizeBoxNumber(boxNumber)
	if requested == "" {
		return nil, fmt.Errorf("box number is required")
	}

	subs, err := c.GetSubscriptionContext(ctx, requested)
	if err != nil {
		return nil, err
	}

	var exact []Subscription
	for _, sub := range subs {
		if matchBoxNumber(requested, sub.BoxNumber__c) {
			exact = append(exact, sub)
		}
	}

	if len(exact) == 0 {
		return nil, fmt.Errorf("box number %q not found", boxNumber)
	}
	if len(exact) > 1 {
		return nil, fmt.Errorf("multiple subscriptions found for box number %q", boxNumber)
	}

	return &exact[0], nil
}

func normalizeSubscriptionNumber(number string) string {
	if number == "" {
		return ""
	}

	number = strings.ToLower(number)
	var b strings.Builder
	b.Grow(len(number))

	for _, r := range number {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}

	return b.String()
}

func normalizeBoxNumber(number string) string {
	number = strings.ToUpper(strings.TrimSpace(number))
	if number == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(number))
	for _, r := range number {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func boxSuffix(box string) string {
	parts := strings.Split(strings.TrimSpace(box), ":")
	return parts[len(parts)-1]
}

func matchBoxNumber(requestedNormalized, actual string) bool {
	actualNormalized := normalizeBoxNumber(actual)
	if actualNormalized == "" {
		return false
	}
	if requestedNormalized == actualNormalized {
		return true
	}

	return requestedNormalized == normalizeBoxNumber(boxSuffix(actual))
}
