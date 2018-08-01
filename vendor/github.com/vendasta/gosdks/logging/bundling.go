package logging

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"context"

	"github.com/vendasta/gosdks/util"
)

// WithBundling runs the given function with a context set up for log bundling to the given stream
// The status code for the log bundle for non-nil errors is computed as util.FromError(err).HttpCode()
// WithBundling() runs the given function synchronously and returns any error from it.
//
// path is used as the URL in the bundled request log, set it to a unique identifier (even for non-request-based code)
// stream is the log-stream to send logs to; e.g. Pubsub, Request, etc.
// Example usage for bundling logging for an executive report cron job:
//
// logging.WithBundling(ctx, "/cron/executive-report/weekly", Background, func(ctx context.Context) error {
//   // do your work here, using the ctx which was passed in
//   // return a util.ServiceError if things fail
// })
func WithBundling(ctx context.Context, path string, stream Stream, work func(context.Context) error) error {
	ctx, requestData := newRequest(ctx, GetLogger().RequestID())
	bundleID := GetLogger().RequestID()
	bundler := getBundler()
	urlObj, _ := url.Parse(path)

	start := time.Now()
	workError := work(ctx)
	end := time.Now()

	latency := end.Sub(start)
	resp := &loggedResponse{status: 200}
	req := &http.Request{URL: urlObj, Method: strings.ToUpper(string(stream))}

	if workError != nil {
		resp.status = util.FromError(workError).HTTPCode()
	}

	ctx = bundler.applyBundlingMetadata(ctx, bundleID, requestData, req, resp, latency)
	logRequest(ctx, requestData, stream)
	return workError
}
