package logging

import (
	"context"
	"net/http"
	"time"

	gce_metadata "cloud.google.com/go/compute/metadata"
	"google.golang.org/grpc/metadata"
)

// gceBundlerOverride will force `getBundler` to return a `GoogleContainerEngineLogBundler`.
// This should only be used for unit testing the GCE log bundler.
var gceBundlerOverride bool

func getBundler() logBundler {
	if gce_metadata.OnGCE() {
		return &GoogleContainerEngineLogBundler{}
	}
	return &NonBundlingLogBundler{}
}

// LogBundler bundles log messages together by surrounding them with a start and end function.
type logBundler interface {
	applyBundlingMetadata(ctx context.Context, bundleID string, rd *requestData,
		request *http.Request, response *loggedResponse, latency time.Duration) context.Context
}

// applyRequestMetadata sets request metadata on a requestData object
func applyRequestMetadata(rd *requestData, request *http.Request, response *loggedResponse, latency time.Duration) {
	if request != nil {
		rd.HTTPRequest.Request.URL = request.URL
		rd.HTTPRequest.Request.Method = request.Method
		rd.HTTPRequest.RemoteIP = request.RemoteAddr
	}

	if response != nil {
		rd.HTTPRequest.Status = int(response.status)
		rd.HTTPRequest.ResponseSize = int64(response.length)
	}

	if latency != 0 {
		rd.HTTPRequest.Latency = latency
	}

	rd.HTTPRequest.LocalIP = "127.0.0.1"
}

// GoogleContainerEngineLogBundler ensures log messages are bundled under a single, collapsible request in cloud logging
type GoogleContainerEngineLogBundler struct {
}

func (g *GoogleContainerEngineLogBundler) applyBundlingMetadata(ctx context.Context, bundleID string, rd *requestData,
	request *http.Request, response *loggedResponse, latency time.Duration) context.Context {
	applyRequestMetadata(rd, request, response, latency)
	rd.Trace = bundleID
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}

// NonBundlingLogBundler satisfies the LogBundler interface but DOES NOT apply any bundling to logs.
type NonBundlingLogBundler struct {
}

func (n *NonBundlingLogBundler) applyBundlingMetadata(ctx context.Context, bundleID string, rd *requestData,
	request *http.Request, response *loggedResponse, latency time.Duration) context.Context {
	applyRequestMetadata(rd, request, response, latency)
	md, _ := metadata.FromOutgoingContext(ctx)
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}
