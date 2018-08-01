package logging

import (
	"io"
	"net/http"
	"time"

	"net/url"

	gce_metadata "cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	"github.com/vendasta/gosdks/util"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
)

// Interceptor provides logging/tracing for incoming gRPC requests
func Interceptor() grpc.UnaryServerInterceptor {
	if gce_metadata.OnGCE() {
		i := &grpcInterceptor{config: configValue, logger: GetLogger()}
		return i.UnaryServerInterceptor
	}
	return PassThroughUnaryServerInterceptor
}

// Interceptor provides logging/tracing for incoming streamed gRPC requests
func StreamInterceptor() grpc.StreamServerInterceptor {
	if gce_metadata.OnGCE() {
		i := &grpcInterceptor{config: configValue, logger: GetLogger()}
		return i.StreamServerInterceptor
	}
	return PassThroughStreamServerInterceptor
}

// Deprecated: Use opencensus
// ClientInterceptor should be used for outgoing gRPC requests.
//
// Should be provided as a dial option on creation of a gRPC transport with grpc.UnaryInterceptor(logging.ClientInterceptor())
func ClientInterceptor() grpc.UnaryClientInterceptor {
	if gce_metadata.OnGCE() {
		i := &grpcInterceptor{config: configValue, logger: GetLogger()}
		return i.UnaryClientInterceptor
	}
	return PassThroughUnaryClientInterceptor
}

// Deprecated: Use opencensus
// ClientStreamInterceptor should be used for outgoing gRPC stream requests.
func ClientStreamInterceptor() grpc.StreamClientInterceptor {
	if gce_metadata.OnGCE() {
		i := &grpcInterceptor{config: configValue, logger: GetLogger()}
		return i.StreamClientInterceptor
	}
	return PassThroughStreamClientInterceptor
}

type grpcInterceptor struct {
	config *config
	logger Logger
}

// PassThroughUnaryServerInterceptor provides an empty incoming interceptor.
func PassThroughUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	logGRPCError(ctx, err)
	return resp, err
}

// PassThroughUnaryServerInterceptor provides an empty incoming interceptor.
func PassThroughStreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, stream)
	logGRPCError(stream.Context(), err)
	return err
}

// PassThroughUnaryClientInterceptor provides an empty outgoing interceptor.
func PassThroughUnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(ctx, method, req, reply, cc, opts...)
}

// PassThroughStreamClientInterceptor provides an empty outgoing stream interceptor.
func PassThroughStreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(ctx, desc, cc, method, opts...)
}

// UnaryServerInterceptor provides an an incoming interceptor for logging/tracing.
func (g *grpcInterceptor) UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	ctx, rd := g.preRequestHook(ctx, info.FullMethod)

	resp, err := handler(ctx, req)

	responseSize := 0
	respProto, ok := resp.(proto.Message)
	if resp != nil && ok {
		responseSize = proto.Size(respProto)
	}

	g.postRequestHook(ctx, info.FullMethod, responseSize, err, rd)

	return resp, err
}

func (g *grpcInterceptor) StreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	ctx, rd := g.preRequestHook(stream.Context(), info.FullMethod)

	wrapped := WrapServerStream(stream)
	wrapped.WrappedContext = ctx
	err := handler(srv, wrapped)

	// report
	g.postRequestHook(ctx, info.FullMethod, 0, err, rd)

	return err
}

func (g *grpcInterceptor) preRequestHook(ctx context.Context, method string) (context.Context, *requestData) {
	// Attach Request Information and start bundling logs
	requestID := g.logger.RequestID()
	ctx, rd := newRequest(ctx, requestID)

	return ctx, rd

}

func (g *grpcInterceptor) postRequestHook(ctx context.Context, path string, responseSize int, err error, rd *requestData) {
	end := time.Now().UTC()
	statusCode := http.StatusOK
	if err != nil {
		statusCode = util.FromError(err).HTTPCode()
		logGRPCError(ctx, err)
	}
	traceID := rd.requestID
	s := trace.FromContext(ctx)
	if s != nil {
		traceID = s.SpanContext().TraceID.String()
	}

	rd.HTTPRequest.Request.URL = &url.URL{Path: path}
	rd.HTTPRequest.Request.Method = "POST"
	rd.HTTPRequest.Status = int(statusCode)
	rd.HTTPRequest.ResponseSize = int64(responseSize)
	rd.HTTPRequest.LocalIP = "127.0.0.1"
	rd.HTTPRequest.RemoteIP = "127.0.0.1"
	rd.HTTPRequest.Latency = end.Sub(rd.startTime)
	rd.Trace = traceID

	fillInFromGRPCMetadata(ctx, rd)
	logRequest(ctx, rd, Request)
}

func logGRPCError(ctx context.Context, err error) {
	if err != nil {
		errType := util.FromError(err).ErrorType().String()
		if util.FromError(err).HTTPCode() >= 500 {
			Errorf(ctx, "Error serving request: %s: %s", errType, err.Error())
		} else {
			Warningf(ctx, "Error serving request: %s: %s", errType, err.Error())
		}
	}
}

func fillInFromGRPCMetadata(ctx context.Context, r *requestData) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
	r.HTTPRequest.Request.Header = http.Header(md)
}

// Deprecated: Use opencensus
// UnaryServerInterceptor provides an an outgoing interceptor for logging/tracing.
func (g *grpcInterceptor) UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx, span := trace.StartSpan(ctx, method)
	defer span.End()
	err := invoker(ctx, method, req, reply, cc, opts...)
	return err
}

// Deprecated: Use opencensus
// StreamClientInterceptor provides an an outgoing stream interceptor for logging/tracing.
func (g *grpcInterceptor) StreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	ctx, span := trace.StartSpan(ctx, method)

	cs, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		span.End()
		return nil, err
	}

	return &monitoredClientStream{cs, span}, nil
}

// monitoredClientStream wraps grpc.ClientStream allowing each Sent/Recv of message to increment counters.
type monitoredClientStream struct {
	grpc.ClientStream
	span *trace.Span
}

func (s *monitoredClientStream) SendMsg(m interface{}) error {
	return s.ClientStream.SendMsg(m)
}

func (s *monitoredClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err == io.EOF {
		s.span.End()
	} else {
		s.span.End()
	}
	return err
}

// HTTPStatusFromCode returns an http status code from a gRPC code.
func HTTPStatusFromCode(code codes.Code) int32 {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

	grpclog.Printf("Unknown gRPC error code: %v", code)
	return http.StatusInternalServerError
}

// WrappedServerStream is a thin wrapper around grpc.ServerStream that allows modifying context.
type WrappedServerStream struct {
	grpc.ServerStream
	// WrappedContext is the wrapper's own Context. You can assign it.
	WrappedContext context.Context
}

// Context returns the wrapper's WrappedContext, overwriting the nested grpc.ServerStream.Context()
func (w *WrappedServerStream) Context() context.Context {
	return w.WrappedContext
}

// WrapServerStream returns a ServerStream that has the ability to overwrite context.
func WrapServerStream(stream grpc.ServerStream) *WrappedServerStream {
	if existing, ok := stream.(*WrappedServerStream); ok {
		return existing
	}
	return &WrappedServerStream{ServerStream: stream, WrappedContext: stream.Context()}
}
