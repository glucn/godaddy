package http

import (
	"context"
	"io"
	"net/http"
)

// Interface holds all the functions
type Interface interface {
	Call(ctx context.Context, method string, url string, body io.Reader, authorization string, contentType string, urlParams []URLParam) (*http.Response, error)
}
