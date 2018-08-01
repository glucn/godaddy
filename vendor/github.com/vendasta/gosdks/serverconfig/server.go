package serverconfig

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/cockroachdb/cmux"
	"github.com/vendasta/gosdks/logging"
	"google.golang.org/grpc"
)

// StartAndListenServer given a grpc.Server and an httpMux will serve on the given port and multiplex the traffic based
// on content type and block until the server has been indicated to be aborted.
func StartAndListenServer(ctx context.Context, grpcServer *grpc.Server, httpMux http.Handler, port int) {
	shutdown := StartServer(ctx, grpcServer, httpMux, port)

	// Block until we receive abort signal.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	shutdown()
}

// StartServer given a grpc.Server and an httpMux will serve on the given port and multiplex the traffic based
// on content type.
func StartServer(ctx context.Context, grpcServer *grpc.Server, httpMux http.Handler, port int) Shutdown {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port));
	if err != nil {
		logging.Criticalf(ctx, "Error starting HTTP Server: %s", err.Error())
		os.Exit(-1)
	}
	mux := cmux.New(lis)
	grpcL := mux.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := mux.Match(cmux.Any())

	logging.Infof(ctx, "Running server on port %d...", port)

	httpServer := &http.Server{
		Handler: httpMux,
	}
	go grpcServer.Serve(grpcL)
	go httpServer.Serve(httpL)
	go mux.Serve()

	return func() {
		// Shutdown Server
		logging.Infof(ctx, "Received abort. Shutting down server...")
		grpcServer.GracefulStop()
		httpServer.Close()
		logging.Infof(ctx, "Server shutdown.")
	}
}

// Shutdown when called will end the running http/grpc servers
type Shutdown func()
