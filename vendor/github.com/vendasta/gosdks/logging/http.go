package logging

import (
	"net/http"

	"time"

	"context"
)

func newLoggedResponse(w http.ResponseWriter) *loggedResponse {
	return &loggedResponse{w, 200, 0}
}

type loggedResponse struct {
	http.ResponseWriter
	status int
	length int
}

func (l *loggedResponse) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func (l *loggedResponse) Write(b []byte) (n int, err error) {
	n, err = l.ResponseWriter.Write(b)
	l.length += n
	return
}

// HTTPMiddleware provides logging/tracing for incoming http requests.
func HTTPMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		ctx, requestData := newRequest(request.Context(), GetLogger().RequestID())
		request = request.WithContext(ctx)

		response := newLoggedResponse(w)

		start := time.Now()
		h.ServeHTTP(response, request)
		end := time.Now()

		bundleID := GetLogger().RequestID()

		logRequestWithBundling(ctx, bundleID, requestData, request, response, end.Sub(start))
	})
}

func logRequestWithBundling(ctx context.Context, bundleID string, requestData *requestData,
	request *http.Request, response *loggedResponse, latency time.Duration) {
	bundler := getBundler()
	ctx = bundler.applyBundlingMetadata(ctx, bundleID, requestData, request, response, latency)
	logRequest(ctx, requestData, Request)
}
