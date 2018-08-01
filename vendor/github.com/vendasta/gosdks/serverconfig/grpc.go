package serverconfig

import (
	"fmt"
	"math"
	"net"
	"runtime/debug"

	"log"

	"github.com/vendasta/gosdks/logging"
	"github.com/vendasta/gosdks/util"
	"go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// CreateGrpcServer creates a basic GRPC Server with the specified interceptors
func CreateGrpcServer(interceptors ...grpc.UnaryServerInterceptor) *grpc.Server {
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: "repcore-prod",
	})
	if err != nil {
		log.Printf("Error creating stackdriver exporter %s", err.Error())
	} else {
		view.RegisterExporter(exporter)
		trace.RegisterExporter(exporter)
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(0.01)})
	}

	interceptors = append(interceptors, RecoveryInterceptor)
	s := grpc.NewServer(
		grpc.UnaryInterceptor(ChainUnaryServer(interceptors...)),
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.MaxConcurrentStreams(math.MaxInt32),
	)
	return s
}

// StartGrpcServer starts a new server to handle GRPC requests
func StartGrpcServer(server *grpc.Server, port int) error {
	var lis net.Listener
	var err error

	if lis, err = net.Listen("tcp", fmt.Sprintf(":%d", port)); err != nil {
		logging.Errorf(context.Background(), "Error creating GRPC listening socket: %s", err.Error())
		return err
	}

	//The following call blocks until an error occurs
	return server.Serve(lis)
}

// RecoveryInterceptor will recover from request level panics and return an Internal error.
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.Criticalf(ctx, "Recovered from panic: %s", debug.Stack())
			err = util.ToGrpcError(util.Error(util.Internal, "An unexpected error occured"))
		}
	}()
	return handler(ctx, req)
}

// NoAuthInterceptor satisfies the GRPCInterceptor but does not actually check any auth
func NoAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

// ChainUnaryServer combines multiple grpc.UnaryServerInterceptor into a single grpc.UnaryServerInterceptor (required by GRPC)
func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		buildChain := func(current grpc.UnaryServerInterceptor, next grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return current(currentCtx, currentReq, info, next)
			}
		}
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildChain(interceptors[i], chain)
		}
		return chain(ctx, req)
	}
}
