package main

import (
	"context"
	"net/http"

	"encoding/json"
	"github.com/glucn/godaddy/internal/godaddy"
	httpService "github.com/glucn/godaddy/internal/http"
	"github.com/vendasta/gosdks/logging"
	"golang.org/x/sync/errgroup"
	"sync"
	"github.com/vendasta/gosdks/serverconfig"
	"google.golang.org/grpc"
)

const (
	APP_NAME = "godaddy"
	httpPort = 11001
)

func main() {
	ctx := context.Background()
	//env := config.CurEnv()

	httpClient := httpService.NewService(&http.Client{})

	godaddyService := godaddy.NewService(httpClient)

	//Start Healthz and Debug HTTP API Server
	healthz := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)

	mux.HandleFunc("/domain-availability", func(w http.ResponseWriter, r *http.Request) {
		domain := "slacknotification.biz"
		available, price, err := godaddyService.GetDomainAvailabilityAndPrice(ctx, domain)

		type response struct {
			Available bool  `json:"available"`
			Price     int64 `json:"price"`
		}
		resp := response{Available: available, Price: price}

		jsonResp, err := json.Marshal(resp)
		if err != nil {
			logging.Errorf(ctx, "Failed to marshal response %#v to json", resp)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)

	})

	mux.HandleFunc("/purchase-domain", func(w http.ResponseWriter, r *http.Request) {
		domain := "Crystalisland.io"
		contact := godaddy.Contact{
			AddressMailing: godaddy.AddressMailing{
				Address1:   "123",
				City:       "Saskatoon",
				Country:    "CA",
				PostalCode: "S7S1N5",
				State:      "SK",
			},
			Email:     "glu+test@vendasta.com",
			NameFirst: "Gary",
			NameLast:  "Lu",
			Phone:     "+123.4511111111",
		}
		consent := godaddy.Consent{
			AgreedAt:      "",
			AgreedBy:      "",
			AgreementKeys: []string{"DNRA"},
		}
		err := godaddyService.PurchaseDomain(ctx, domain, contact, consent)

		if err != nil {
			logging.Errorf(ctx, "Error: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/domain-suggest", func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			Domain string `json:"domain"`
		}
		req := request{}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logging.Errorf(ctx, "Failed to parse request: %#v", r)
			http.Error(w, "Error processing request, expected {domain: string}", http.StatusBadRequest)
			return
		}

		domain := req.Domain

		sDomains, err := godaddyService.GetDomainSuggestions(ctx, domain)

		if err != nil {
			logging.Errorf(ctx, "Error getting domain suggestion for %s: %s", domain, err.Error())
			http.Error(w, "Error getting domain suggestion", http.StatusInternalServerError)
			return
		}

		type suggestion struct {
			Domain string `json:"domain"`
			Price  int64  `json:"price"` // price in cent
		}

		type response struct {
			Suggestion []suggestion
		}

		type mutexSuggestions struct {
			mu sync.Mutex
			s  []suggestion
		}

		domainAndPrice := &mutexSuggestions{}
		g, ctx := errgroup.WithContext(ctx)

		for i, d := range sDomains {
			g.Go(func(index int, domain string) func() error {
				return func() error {
					logging.Infof(ctx, "Getting price for %s", domain)
					avail, price, err := godaddyService.GetDomainAvailabilityAndPrice(ctx, domain)
					if err != nil {
						logging.Errorf(ctx, "Error getting domain availability and price for %s: %s", domain, err.Error())
						return err
					}
					if avail {
						domainAndPrice.mu.Lock()
						domainAndPrice.s = append(domainAndPrice.s, suggestion{
							Domain: domain,
							Price:  price,
						})
						domainAndPrice.mu.Unlock()
					}
					return nil
				}
			}(i, d))
		}

		err = g.Wait()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		jsonResp, err := json.Marshal(response{Suggestion: domainAndPrice.s})
		if err != nil {
			logging.Errorf(ctx, "Failed to marshal response %v to json", domainAndPrice)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)

		return

	})

	mux.HandleFunc("/list-dns", func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			Domain string `json:"domain"`
		}
		req := request{}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logging.Errorf(ctx, "Failed to parse request: %#v", r)
			http.Error(w, "Error processing request, expected {domain: string}", http.StatusBadRequest)
			return
		}

		domain := req.Domain

		type mutexDNSRecords struct {
			mu sync.Mutex
			r  []godaddy.DNSRecord
		}

		dnsRecords := &mutexDNSRecords{}
		g, ctx := errgroup.WithContext(ctx)

		for _, t := range godaddy.DNSTypes {
			g.Go(func(DNSType string) func() error {
				return func() error {
					records, err := godaddyService.GetDNSRecords(ctx, domain, DNSType)
					if err != nil {
						logging.Errorf(ctx, "Error getting DNS records for domain %s, type %s: %s", domain, DNSType, err.Error())
						return err
					}

					logging.Infof(ctx, "DNS records of type %s: %s", DNSType, records)
					dnsRecords.mu.Lock()
					dnsRecords.r = append(dnsRecords.r, records...)
					dnsRecords.mu.Unlock()
					return nil
				}
			}(t))
		}
		err = g.Wait()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		type response struct {
			DNSRecords []godaddy.DNSRecord `json:"records"`
		}

		jsonResp, err := json.Marshal(response{DNSRecords: dnsRecords.r})
		if err != nil {
			logging.Errorf(ctx, "Failed to marshal response %v to json", dnsRecords)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
		return
	})

	mux.HandleFunc("/put-dns", func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			Domain string `json:"domain"`
			Type string `json:"type"`
			Name string `json:"name"`
			Data string `json:"data"`
			TTL int64 `json:"ttl"`
		}
		req := request{}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			logging.Errorf(ctx, "Failed to parse request: %#v", r)
			http.Error(w, "Error processing request, expected {domain: string}", http.StatusBadRequest)
			return
		}

		domain := req.Domain
		dnsRecord := godaddy.DNSRecord{
			Type: req.Type,
			Name: req.Name,
			Data: req.Data,
			TTL: req.TTL,
		}

		// Add validations

		err = godaddyService.PutDNSRecord(ctx, domain, dnsRecord)
		if err != nil {
			logging.Errorf(ctx, "Error putting DNS records for domain %s: %s", domain, err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	})

	logging.Infof(ctx, "Starting HTTP server...")
	serverconfig.StartAndListenServer(ctx, grpc.NewServer(), mux, httpPort)

	//for i := 0; i<100; i++ {
	//	//domain := randomdata.FirstName(randomdata.RandomGender) + randomdata.LastName() + ".ca"
	//	domain := randomdata.SillyName() + ".biz"
	//	available, price, _ := godaddyService.GetDomainAvailabilityAndPrice(ctx, domain)
	//
	//	logging.Infof(ctx, "domain %s, available %t, price %d", domain, available, price)
	//	time.Sleep(time.Second)
	//}

	//tlds, err := godaddyService.ListTLDs(ctx)
	//
	//if err != nil {
	//	logging.Errorf(ctx, "Error listing TLDs: %s", err.Error())
	//	return
	//}
	//
	//for _, tld := range tlds {
	//	required, err := godaddyService.GetPurchaseAgreement(ctx, tld)
	//	if err != nil {
	//		logging.Errorf(ctx, "Error getting purchase schema for %s: %s", tld, err.Error())
	//		return
	//	}
	//	logging.Infof(ctx, "TLD %s agreement: %s", tld, required)
	//	time.Sleep(time.Second)
	//}
	//return
}
