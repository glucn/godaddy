package http

import (
	"context"
	"fmt"
	"github.com/vendasta/gosdks/logging"
	"github.com/vendasta/gosdks/util"
	"github.com/vendasta/gosdks/validation"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// Service is an http
type Service struct {
	httpClient *http.Client
}

// URLParam holds a parameter of HTTP call
type URLParam struct {
	Key   string
	Value string
}

const (
	// ContentTypeJSON is the content type for JSON body
	ContentTypeJSON = "application/json"

	// ContentTypeAtomXML is the content type for atom XML body
	ContentTypeAtomXML = "application/atom+xml"
)

// Error represents an error when calling a service over http
type Error struct {
	Body       string
	StatusCode int
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Body
}

// NewService returns a new implementation of the http service
func NewService(httpClient *http.Client) Interface {
	return &Service{
		httpClient: httpClient,
	}
}

// Call will do an http call given a valid http method (ex: GET, POST, PUT...)
func (s *Service) Call(ctx context.Context, method string, url string, body io.Reader, authorization string, contentType string, urlParams []URLParam) (*http.Response, error) {
	err := validation.NewValidator().
		Rule(validation.StringNotEmpty(url, util.InvalidArgument, "URL must not be empty")).
		Rule(validation.StringNotEmpty(method, util.InvalidArgument, "Method must not be empty")).
		Validate()
	if err != nil {
		logging.Errorf(ctx, "Failed validation when doing http call: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(strings.ToUpper(method), url, body)
	if err != nil {
		logging.Errorf(ctx, "Error creating %s http request with url %s, body %v: %v", method, url, body, err)
		return nil, util.Error(util.Internal, "Error getting http request")
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", contentType)

	if urlParams != nil && len(urlParams) > 0 {
		q := req.URL.Query()

		for i := 0; i < len(urlParams); i++ {
			q.Add(urlParams[i].Key, urlParams[i].Value)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logging.Errorf(ctx, "Error doing %s http request with request %v: %v", method, req, err)
		return nil, util.Error(util.Internal, "Error during http call")
	}

	if resp.StatusCode > 299 {
		logging.Errorf(ctx, "Error doing %s http request with request %v: %v", method, req, err)
		return nil, parseError(resp)
	}

	return resp, nil
}

func parseError(r *http.Response) error {
	body := ""
	if r.Body != nil {
		defer r.Body.Close()
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		body = string(bodyBytes)
	}

	errorBody := fmt.Sprintf("%s: %s", http.StatusText(r.StatusCode), body)
	return &Error{StatusCode: r.StatusCode, Body: errorBody}
}
