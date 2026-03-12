package canalbox

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (c *Client) GetRenewOptionsByBox(ctx context.Context, boxNumber string) (*BoxRenewOptions, error) {
	sub, err := c.getSubscriptionByBoxNumber(ctx, boxNumber)
	if err != nil {
		return nil, err
	}

	options, err := c.GetAvailableRenewOptionsForSubscription(ctx, *sub, RenewOptionsRequest{
		CountryCode:       DefaultCountryCode,
		PaymentMode:       DefaultPaymentMode,
		CurrentFibreOffer: sub.SUB_T_Produit_De_Base__c,
	})
	if err != nil {
		return nil, err
	}

	return &BoxRenewOptions{
		Subscription: *sub,
		Offers:       collectRenewalOffers(options),
	}, nil
}

func (c *Client) GetAvailableRenewOptions(ctx context.Context, pageURI string, req RenewOptionsRequest) (*RenewOptionsResponse, error) {
	ctx = ensureContext(ctx)
	mapping := map[string]any{
		"countryCode":       req.CountryCode,
		"modeDePaiement":    req.PaymentMode,
		"CurrentFibreOffre": req.CurrentFibreOffer,
	}
	if req.BundleStatus != nil {
		mapping["bundleStatus"] = *req.BundleStatus
	}

	params := map[string]any{"mapOfSubInfo": mapping}

	response, err := c.doAuraAction(ctx, pageURI, "AP01_ClasseGenerale", "getAvailableRenewAction", params)
	if err != nil {
		return nil, err
	}

	action, err := firstAction(response)
	if err != nil {
		return nil, err
	}

	var result RenewOptionsResponse
	if err := decodeReturnValue(action.ReturnValue, &result); err != nil {
		return nil, fmt.Errorf("parse renew options: %w", err)
	}

	return &result, nil
}

func (c *Client) GetAvailableRenewOptionsForSubscription(ctx context.Context, sub Subscription, req RenewOptionsRequest) (*RenewOptionsResponse, error) {
	pageURI, err := sub.PageURI()
	if err != nil {
		return nil, err
	}

	return c.GetAvailableRenewOptions(ctx, pageURI, req)
}

func (c *Client) PreviewRenewByBox(ctx context.Context, boxNumber, offerName string, months int) (*RenewPreviewResult, error) {
	req, sub, err := c.buildRenewRequestForBox(ctx, boxNumber, offerName, months)
	if err != nil {
		return nil, err
	}

	return c.PreviewSubscription(ctx, *sub, req)
}

func (c *Client) ActivateRenewByBox(ctx context.Context, boxNumber, offerName string, months int) (*RenewPreviewResult, error) {
	req, sub, err := c.buildRenewRequestForBox(ctx, boxNumber, offerName, months)
	if err != nil {
		return nil, err
	}

	return c.ActivateSubscription(ctx, *sub, req)
}

func (c *Client) PreviewRenew(ctx context.Context, pageURI string, req RenewRequest) (*RenewPreviewResult, error) {
	return c.handlePreviewAndRenew(ctx, pageURI, req, true)
}

func (c *Client) PreviewSubscription(ctx context.Context, sub Subscription, req RenewRequest) (*RenewPreviewResult, error) {
	pageURI, err := sub.PageURI()
	if err != nil {
		return nil, err
	}

	req, err = validateRenewTarget(sub, req)
	if err != nil {
		return nil, err
	}

	return c.PreviewRenew(ctx, pageURI, req)
}

func (c *Client) ActivateRenew(ctx context.Context, pageURI string, req RenewRequest) (*RenewPreviewResult, error) {
	return c.handlePreviewAndRenew(ctx, pageURI, req, false)
}

func (c *Client) ActivateSubscription(ctx context.Context, sub Subscription, req RenewRequest) (*RenewPreviewResult, error) {
	pageURI, err := sub.PageURI()
	if err != nil {
		return nil, err
	}

	req, err = validateRenewTarget(sub, req)
	if err != nil {
		return nil, err
	}

	return c.ActivateRenew(ctx, pageURI, req)
}

func validateRenewTarget(sub Subscription, req RenewRequest) (RenewRequest, error) {
	if req.SubscriptionID == "" {
		req.SubscriptionID = sub.Id
	}
	if req.SubscriptionID != sub.Id {
		return req, fmt.Errorf("renew request subscription id %q does not match subscription %q", req.SubscriptionID, sub.Id)
	}

	if req.ExpectedBoxNumber == "" {
		req.ExpectedBoxNumber = sub.BoxNumber__c
	}
	if !matchBoxNumber(normalizeBoxNumber(req.ExpectedBoxNumber), sub.BoxNumber__c) {
		return req, fmt.Errorf("renew request box number %q does not match subscription box %q", req.ExpectedBoxNumber, sub.BoxNumber__c)
	}

	return req, nil
}

func (c *Client) handlePreviewAndRenew(ctx context.Context, pageURI string, req RenewRequest, isPreview bool) (*RenewPreviewResult, error) {
	ctx = ensureContext(ctx)
	mapOfParams := map[string]any{
		"renewType":        req.RenewType,
		"subId":            req.SubscriptionID,
		"nbPeriodes":       fmt.Sprintf("%d", req.NbPeriodes),
		"paymentMode":      req.PaymentMode,
		"offerCanal":       req.OfferCanal,
		"dollarPayment":    req.DollarPayment,
		"CDFPayment":       req.CDFPayment,
		"dollarRefund":     req.DollarRefund,
		"CDFRefund":        req.CDFRefund,
		"bizaOrderId":      req.BizaOrderID,
		"withRenew":        req.WithRenew,
		"immediateUpgrade": req.ImmediateUpgrade,
	}

	payload := map[string]any{
		"mapOfParams":  mapOfParams,
		"optionsCanal": req.OptionsCanal,
		"isPreview":    isPreview,
	}

	response, err := c.doAuraAction(ctx, pageURI, "AP01_ClasseGenerale", "handlePreviewANDRenew", payload)
	if err != nil {
		return nil, err
	}

	action, err := firstAction(response)
	if err != nil {
		return nil, err
	}

	var result RenewPreviewResult
	if err := decodeReturnValue(action.ReturnValue, &result); err != nil {
		return nil, fmt.Errorf("parse preview/activation result: %w", err)
	}

	return &result, nil
}

func (c *Client) buildRenewRequestForBox(ctx context.Context, boxNumber, offerName string, months int) (RenewRequest, *Subscription, error) {
	if months < 1 {
		return RenewRequest{}, nil, fmt.Errorf("months must be greater than 0")
	}

	info, err := c.GetRenewOptionsByBox(ctx, boxNumber)
	if err != nil {
		return RenewRequest{}, nil, err
	}

	offer, err := findOfferByName(info.Offers, offerName)
	if err != nil {
		return RenewRequest{}, nil, err
	}

	req := RenewRequest{
		SubscriptionID:    info.Subscription.Id,
		ExpectedBoxNumber: boxNumber,
		RenewType:         renewTypeFromLevel(offer.Level),
		NbPeriodes:        months,
		PaymentMode:       DefaultPaymentMode,
		OfferCanal:        offer.Name,
		BizaOrderID:       buildOrderID(info.Subscription.SubscriptionNumber),
		WithRenew:         true,
		ImmediateUpgrade:  true,
	}

	return req, &info.Subscription, nil
}

func collectRenewalOffers(options *RenewOptionsResponse) []RenewalOffer {
	if options == nil {
		return nil
	}

	seen := make(map[string]bool)
	offers := make([]RenewalOffer, 0)

	add := func(name, level string, current bool) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToUpper(name)
		if seen[key] {
			if current {
				for i := range offers {
					if strings.EqualFold(offers[i].Name, name) {
						offers[i].Current = true
						break
					}
				}
			}
			return
		}

		seen[key] = true
		offers = append(offers, RenewalOffer{Name: name, Level: strings.TrimSpace(level), Current: current})
	}

	add(options.CurrentOption.OfferName, options.CurrentOption.Level, true)
	for _, offer := range options.FibreOptions {
		add(offer.OfferName, offer.Level, false)
	}
	for _, offer := range options.BundleOptions {
		add(offer.OfferName, offer.Level, false)
	}

	return offers
}

func findOfferByName(offers []RenewalOffer, offerName string) (*RenewalOffer, error) {
	wanted := strings.TrimSpace(offerName)
	if wanted == "" {
		for i := range offers {
			if offers[i].Current {
				return &offers[i], nil
			}
		}
		if len(offers) > 0 {
			return &offers[0], nil
		}
		return nil, fmt.Errorf("no offer available")
	}

	for i := range offers {
		if strings.EqualFold(offers[i].Name, wanted) {
			return &offers[i], nil
		}
	}

	names := make([]string, 0, len(offers))
	for _, offer := range offers {
		names = append(names, offer.Name)
	}

	return nil, fmt.Errorf("offer %q not available, choose one of: %s", offerName, strings.Join(names, ", "))
}

func renewTypeFromLevel(level string) string {
	level = strings.TrimSpace(level)
	if level == "" {
		return "REABO-1"
	}
	return "REABO-" + level
}

func buildOrderID(subscriptionNumber string) string {
	return fmt.Sprintf("%s-%s", subscriptionNumber, time.Now().Format("20060102150405"))
}
