package canalbox

import "fmt"

const defaultSubscriptionListPageURI = "/PortailDistributeur/s/subscription/Zuora__Subscription__c/Default?tabset-c5778=2"

const (
	DefaultCountryCode = "RW"
	DefaultPaymentMode = "Cash"
)

type Subscription struct {
	Name                      string          `json:"Name"`
	Id                        string          `json:"Id"`
	BoxNumber__c              string          `json:"BoxNumber__c"`
	ExpectedRenewalDate__c    string          `json:"ExpectedRenewalDate__c"`
	Zuora__Account__c         string          `json:"Zuora__Account__c"`
	Zuora__CustomerAccount__c string          `json:"Zuora__CustomerAccount__c"`
	SUB_T_Produit_De_Base__c  string          `json:"SUB_T_Produit_De_Base__c"`
	SubscriptionNumber        string          `json:"SubscriptionNumber"`
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

type RenewOptionsRequest struct {
	CountryCode       string
	PaymentMode       string
	CurrentFibreOffer string
	BundleStatus      *string
}

type RenewActionOption struct {
	OfferName string `json:"offerName"`
	Level     string `json:"level"`
}

type RenewalOffer struct {
	Name    string
	Level   string
	Current bool
}

type BoxRenewOptions struct {
	Subscription Subscription
	Offers       []RenewalOffer
}

type RenewOptionsResponse struct {
	BundleOptions []RenewActionOption `json:"bundleOptions"`
	CurrentOption RenewActionOption   `json:"currentOption"`
	FibreOptions  []RenewActionOption `json:"fibreOptions"`
}

type RenewRequest struct {
	SubscriptionID    string
	ExpectedBoxNumber string
	RenewType         string
	NbPeriodes        int
	PaymentMode       string
	OfferCanal        string
	BizaOrderID       string
	WithRenew         bool
	ImmediateUpgrade  bool
	DollarPayment     *float64
	CDFPayment        *float64
	DollarRefund      *float64
	CDFRefund         *float64
	OptionsCanal      []any
}

type PreviewResultDetail struct {
	Invoices []PreviewInvoice `json:"invoices"`
}

type PreviewInvoice struct {
	TargetDate       string               `json:"targetDate"`
	Amount           float64              `json:"amount"`
	AmountWithoutTax float64              `json:"amountWithoutTax"`
	TaxAmount        float64              `json:"taxAmount"`
	InvoiceItems     []PreviewInvoiceItem `json:"invoiceItems"`
}

type PreviewInvoiceItem struct {
	ServiceStartDate string  `json:"serviceStartDate"`
	ServiceEndDate   string  `json:"serviceEndDate"`
	AmountWithoutTax float64 `json:"amountWithoutTax"`
	TaxAmount        float64 `json:"taxAmount"`
	ChargeName       string  `json:"chargeName"`
	RatePlanName     string  `json:"ratePlanName"`
	ProductName      string  `json:"productName"`
	Bandwidth        string  `json:"bandwidth"`
}

type ActivationReason struct {
	Message string `json:"message"`
}

type RenewPreviewResult struct {
	Success        bool                 `json:"success"`
	Reasons        []ActivationReason   `json:"reasons"`
	PreviewResult  *PreviewResultDetail `json:"previewResult"`
	ConversionRate float64              `json:"conversionRate"`
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
