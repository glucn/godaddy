package godaddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	httpService "github.com/glucn/godaddy/internal/http"
	"github.com/vendasta/gosdks/logging"
	"github.com/vendasta/gosdks/util"
	"io/ioutil"
	"net/http"
)

const (
	domainsAvailableURL          = "https://api.godaddy.com/v1/domains/available"
	purchaseDomainURL            = "https://api.ote-godaddy.com/v1/domains/purchase"
	domainSuggestURL             = "https://api.ote-godaddy.com/v1/domains/suggest"
	listTLDsURL                  = "https://api.ote-godaddy.com/v1/domains/tlds"
	getPurchaseSchemaURLTemplate = "https://api.ote-godaddy.com/v1/domains/purchase/schema/%s"
	getPurchaseAgreementURL      = "https://api.ote-godaddy.com/v1/domains/agreements"
	getDNSRecordsURLTemplate     = "https://api.ote-godaddy.com/v1/domains/%s/records/%s"
	putDNSRecordURLTemplate      = "https://api.ote-godaddy.com/v1/domains/%s/records/%s/%s"

	auth = ""
)

var DNSTypes = []string{"A", "AAAA", "CNAME", "MX", "NS", "SOA", "SRV", "TXT"}

// Service is a service for GoDaddy APIs
type Service struct {
	httpClient httpService.Interface
}

// NewService returns a new implementation of the service for the service provider
func NewService(hc httpService.Interface) Interface {
	return &Service{
		httpClient: hc,
	}
}

func (s *Service) GetDomainAvailabilityAndPrice(ctx context.Context, domain string) (bool, int64, error) {
	param := []httpService.URLParam{
		{
			Key:   "domain",
			Value: domain,
		},
	}
	res, err := s.httpClient.Call(ctx, http.MethodGet, domainsAvailableURL, nil, auth, "", param)

	if err != nil {
		logging.Errorf(ctx, "Error calling /v1/domains/available: %s", err.Error())
		return false, 0, util.Error(util.Internal, "Error getting domain availability")
	}

	type response struct {
		Available bool   `json:"available"`
		Currency  string `json:"currency"`
		Price     int64  `json:"price"`
		//Definitive bool   `json:"definitive"`
		//Domain     string `json:"domain"`
		//Period     int64  `json:"period"`
	}

	body := &response{}
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logging.Errorf(ctx, "Error reading response body %v: %v", res, err)
		return false, 0, util.Error(util.Internal, "Error reading response body")
	}
	json.Unmarshal(buf, body)

	return body.Available, body.Price, nil
}

func (s *Service) PurchaseDomain(ctx context.Context, domain string, contact Contact, consent Consent) error {

	type purchaseDomainBody struct {
		Consent           Consent `json:"consent"`
		ContactAdmin      Contact `json:"contactAdmin"`
		ContactBilling    Contact `json:"contactBilling"`
		ContactRegistrant Contact `json:"contactRegistrant"`
		ContactTech       Contact `json:"contactTech"`
		Domain            string  `json:"domain"`
	}

	bodyData := purchaseDomainBody{
		Consent:           consent,
		ContactAdmin:      contact,
		ContactBilling:    contact,
		ContactRegistrant: contact,
		ContactTech:       contact,
		Domain:            domain,
	}

	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(bodyData)

	_, err := s.httpClient.Call(ctx, http.MethodPost, purchaseDomainURL, body, auth, httpService.ContentTypeJSON, nil)

	if err != nil {
		logging.Errorf(ctx, "Error calling /v1/domains/purchase: %s", err.Error())
		return util.Error(util.Internal, "Error purchasing domain")
	}

	return nil
}

func (s *Service) GetDomainSuggestions(ctx context.Context, domain string) ([]string, error) {
	param := []httpService.URLParam{
		{
			Key:   "query",
			Value: domain,
		},
		{
			Key:   "limit",
			Value: "10",
		},
	}

	res, err := s.httpClient.Call(ctx, http.MethodGet, domainSuggestURL, nil, auth, "", param)

	if err != nil {
		logging.Errorf(ctx, "Error calling %s: %s", domainSuggestURL, err.Error())
		return nil, util.Error(util.Internal, "Error getting domain suggestion")
	}

	type response struct {
		Domain string `json:"domain"`
	}

	body := make([]response, 0)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logging.Errorf(ctx, "Error reading response body %v: %v", res, err)
		return nil, util.Error(util.Internal, "Error reading response body")
	}
	json.Unmarshal(buf, &body)

	resp := make([]string, len(body))
	for i, d := range body {
		resp[i] = d.Domain
	}
	return resp, nil
}

func (s *Service) GetPurchaseSchema(ctx context.Context, tld string) ([]string, error) {
	url := fmt.Sprintf(getPurchaseSchemaURLTemplate, tld)
	res, err := s.httpClient.Call(ctx, http.MethodGet, url, nil, auth, httpService.ContentTypeJSON, nil)
	if err != nil {
		logging.Errorf(ctx, "Error calling %s: %s", url, err.Error())
		return nil, util.Error(util.Internal, "Error getting purchase schema")
	}

	type response struct {
		Required []string `json:"required"`
	}

	body := &response{}
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logging.Errorf(ctx, "Error reading response body %v: %v", res, err)
		return nil, util.Error(util.Internal, "Error reading response body")
	}
	json.Unmarshal(buf, body)

	return body.Required, nil
}

func (s *Service) ListTLDs(ctx context.Context) ([]string, error) {
	res, err := s.httpClient.Call(ctx, http.MethodGet, listTLDsURL, nil, auth, "", nil)

	if err != nil {
		logging.Errorf(ctx, "Error calling %s: %s", listTLDsURL, err.Error())
		return nil, util.Error(util.Internal, "Error listing supported TLDs")
	}

	type response struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	body := make([]response, 0)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logging.Errorf(ctx, "Error reading response body %v: %v", res, err)
		return nil, util.Error(util.Internal, "Error reading response body")
	}
	json.Unmarshal(buf, &body)

	resp := make([]string, len(body))
	for i, d := range body {
		resp[i] = d.Name
	}
	return resp, nil
}

func (s *Service) GetPurchaseAgreement(ctx context.Context, tld string) ([]string, error) {
	param := []httpService.URLParam{
		{
			Key:   "tlds",
			Value: tld,
		},
		{
			Key:   "privacy",
			Value: "false",
		},
	}
	res, err := s.httpClient.Call(ctx, http.MethodGet, getPurchaseAgreementURL, nil, auth, "", param)

	if err != nil {
		logging.Errorf(ctx, "Error calling %s: %s", getPurchaseAgreementURL, err.Error())
		return nil, util.Error(util.Internal, "Error getting purchase agreement")
	}

	type response struct {
		AgreementKey string `json:"agreementKey"`
		Content      string `json:"content"`
		Title        string `json:"title"`
		URL          string `json:"url"`
	}

	body := make([]response, 0)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logging.Errorf(ctx, "Error reading response body %v: %v", res, err)
		return nil, util.Error(util.Internal, "Error reading response body")
	}
	json.Unmarshal(buf, &body)

	resp := make([]string, len(body))
	for i, d := range body {
		resp[i] = d.AgreementKey
	}
	return resp, nil
}

func (s *Service) GetDNSRecords(ctx context.Context, domain string, dnsType string) ([]DNSRecord, error) {

	url := fmt.Sprintf(getDNSRecordsURLTemplate, domain, dnsType)

	res, err := s.httpClient.Call(ctx, http.MethodGet, url, nil, auth, "", nil)
	if err != nil {
		logging.Errorf(ctx, "Error calling %s: %s", url, err.Error())
		return nil, util.Error(util.Internal, "Error getting DNS records")
	}

	body := make([]DNSRecord, 0)
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logging.Errorf(ctx, "Error reading response body %v: %v", res, err)
		return nil, util.Error(util.Internal, "Error reading response body")
	}
	json.Unmarshal(buf, &body)

	return body, nil

}

func (s *Service) PutDNSRecord(ctx context.Context, domain string, record DNSRecord) error {
	url := fmt.Sprintf(putDNSRecordURLTemplate, domain, record.Type, record.Name)

	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode([]DNSRecord{record})

	_, err := s.httpClient.Call(ctx, http.MethodPut, url, body, auth, httpService.ContentTypeJSON, nil)

	if err != nil {
		logging.Errorf(ctx, "Error calling %s: %s", url, err.Error())
		return err
	}

	return nil
}
