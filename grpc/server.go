package grpc

import (
	"context"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/go-logr/logr"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/freecache"
	v1 "github.com/tinkerbell/pbnj/api/v1"
	"github.com/tinkerbell/pbnj/grpc/oob"
	"github.com/tinkerbell/pbnj/grpc/persistence"
	"github.com/tinkerbell/pbnj/grpc/rpc"
	"github.com/tinkerbell/pbnj/grpc/taskrunner"
	"github.com/tinkerbell/pbnj/pkg/healthcheck"
	"github.com/tinkerbell/pbnj/pkg/http"
	"github.com/tinkerbell/pbnj/pkg/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server options.
type Server struct {
	repository.Actions
	bmcTimeout time.Duration
	// skipRedfishVersions is a list of Redfish versions to be ignored,
	//
	// When running an action on a BMC, PBnJ will pass the value of the skipRedfishVersions to bmclib
	// which will then ignore the Redfish endpoint completely on BMCs running the given Redfish versions,
	// and will proceed to attempt other drivers like - IPMI/SSH/Vendor API instead.
	//
	// for more information see https://github.com/bmc-toolbox/bmclib#bmc-connections
	skipRedfishVersions []string
}

// ServerOption for setting optional values.
type ServerOption func(*Server)

// WithPersistence sets the log level.
func WithPersistence(repo repository.Actions) ServerOption {
	return func(args *Server) { args.Actions = repo }
}

// WithBmcTimeout sets the timeout for BMC calls.
func WithBmcTimeout(t time.Duration) ServerOption {
	return func(args *Server) { args.bmcTimeout = t }
}

// WithSkipRedfishVersions sets the Redfish versions to ignore for BMC calls.
func WithSkipRedfishVersions(versions []string) ServerOption {
	return func(args *Server) { args.skipRedfishVersions = versions }
}

// RunServer registers all services and runs the server.
func RunServer(ctx context.Context, log logr.Logger, grpcServer *grpc.Server, port string, httpServer *http.Server, opts ...ServerOption) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defaultStore := gokv.Store(freecache.NewStore(freecache.DefaultOptions))

	// instantiate a Repository for task persistence
	repo := &persistence.GoKV{
		Store: defaultStore,
		Ctx:   ctx,
	}

	defaultServer := &Server{
		Actions:    repo,
		bmcTimeout: oob.DefaultBMCTimeout,
	}

	for _, opt := range opts {
		opt(defaultServer)
	}

	taskRunner := &taskrunner.Runner{
		Repository: defaultServer.Actions,
		Ctx:        ctx,
		Dispatcher: taskrunner.NewDispatcher(),
	}
	go taskRunner.Start(ctx)

	ms := rpc.MachineService{
		TaskRunner: taskRunner,
		Timeout:    defaultServer.bmcTimeout,
	}
	v1.RegisterMachineServer(grpcServer, &ms)

	bs := rpc.BmcService{
		TaskRunner:          taskRunner,
		Timeout:             defaultServer.bmcTimeout,
		SkipRedfishVersions: defaultServer.skipRedfishVersions,
	}
	v1.RegisterBMCServer(grpcServer, &bs)

	ds := rpc.DiagnosticService{}
	v1.RegisterDiagnosticServer(grpcServer, &ds)

	ts := rpc.TaskService{
		TaskRunner: taskRunner,
	}
	v1.RegisterTaskServer(grpcServer, &ts)

	grpc_prometheus.Register(grpcServer)

	hc := healthcheck.NewHealthChecker()
	grpc_health_v1.RegisterHealthServer(grpcServer, hc)

	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	httpServer.WithTaskRunner(taskRunner)
	reflection.Register(grpcServer)

	go func() {
		err := httpServer.Run()
		if err != nil {
			log.Error(err, "failed to serve http")
			os.Exit(1) //nolint:revive // removing deep-exit requires a significant refactor
		}
	}()

	// graceful shutdowns
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		for range sigChan {
			log.Info("sig received, shutting down PBnJ")
			grpcServer.GracefulStop()
			<-ctx.Done()
		}
	}()

	go func() {
		<-ctx.Done()
		log.Info("ctx cancelled, shutting down PBnJ")
		grpcServer.GracefulStop()
	}()

	log.Info("starting PBnJ gRPC server")
	return grpcServer.Serve(listen)
}
