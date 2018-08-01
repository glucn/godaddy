package godaddy

import "context"

type AddressMailing struct {
	Address1   string `json:"address1"`
	City       string `json:"city"`
	Country    string `json:"country"`
	PostalCode string `json:"postalCode"`
	State      string `json:"state"`
}

type Contact struct {
	AddressMailing AddressMailing `json:"addressMailing"`
	Email          string         `json:"email"`
	NameFirst      string         `json:"nameFirst"`
	NameLast       string         `json:"nameLast"`
	Phone          string         `json:"phone"`
}

type Consent struct {
	AgreedAt      string   `json:"agreedAt"`
	AgreedBy      string   `json:"agreedBy"`
	AgreementKeys []string `json:"agreementKeys"`
}

type DNSRecord struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	TTL  int64  `json:"ttl"`
}

// Interface holds the GoDaddy APIs
type Interface interface {
	GetDomainAvailabilityAndPrice(ctx context.Context, domain string) (bool, int64, error)
	PurchaseDomain(ctx context.Context, domain string, contact Contact, consent Consent) error
	GetDomainSuggestions(ctx context.Context, domain string) ([]string, error)
	GetPurchaseSchema(ctx context.Context, tld string) ([]string, error)
	ListTLDs(ctx context.Context) ([]string, error)
	GetPurchaseAgreement(ctx context.Context, tld string) ([]string, error)
	GetDNSRecords(ctx context.Context, domain string, dnsType string) ([]DNSRecord, error)
	PutDNSRecord(ctx context.Context, domain string, record DNSRecord) error
}
