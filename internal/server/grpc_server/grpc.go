package grpc_server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	_ "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/okieraised/monitoring-agent/internal/config"
	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

func getGRPCPort() int {
	port := viper.GetInt(config.AgentGRPCPort)
	if port <= 0 {
		return constants.AgentDefaultMonitoringPort
	}
	return port
}

// NewGRPCServer starts the server, blocks until ctx is done, then graceful-stops.
func NewGRPCServer(ctx context.Context, registerServices func(s *grpc.Server)) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", getGRPCPort()))
	if err != nil {
		wErr := errors.Wrap(err, "failed to listen")
		log.Default().Error(wErr.Error())
		return wErr
	}
	defer func(lis net.Listener) {
		cErr := lis.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(lis)

	// Interceptor chains
	var unaryInts []grpc.UnaryServerInterceptor
	var streamInts []grpc.StreamServerInterceptor

	unaryInts = append(unaryInts,
		grpc_recovery.UnaryServerInterceptor(
			grpc_recovery.WithRecoveryHandler(
				func(err any) error {
					log.Default().Error(fmt.Sprintf("panic recovered: %v", err))
					return status.Errorf(status.Code(grpc.ErrServerStopped), "internal server error")
				},
			)),
	)
	streamInts = append(streamInts,
		grpc_recovery.StreamServerInterceptor(
			grpc_recovery.WithRecoveryHandler(
				func(err any) error {
					log.Default().Error(fmt.Sprintf("panic recovered: %v", err))
					return status.Errorf(status.Code(grpc.ErrServerStopped), "internal server error")
				},
			)),
	)

	var serverOpts []grpc.ServerOption

	if viper.GetString(config.AgentTLSCertFile) != "" && viper.GetString(config.AgentTLSKeyFile) != "" {
		cert, err := tls.LoadX509KeyPair(viper.GetString(config.AgentTLSCertFile), viper.GetString(config.AgentTLSKeyFile))
		if err != nil {
			wErr := errors.Wrap(err, "failed to load server cert file")
			return wErr
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		if viper.GetString(config.AgentTLSClientCAFile) != "" {
			caBytes, err := os.ReadFile(viper.GetString(config.AgentTLSClientCAFile))
			if err != nil {
				wErr := errors.Wrap(err, "failed to read client CA file")
				return wErr
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caBytes) {
				wErr := errors.Wrap(err, "failed to append client CA to pool")
				return wErr
			}
			tlsCfg.ClientCAs = pool
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		}
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	serverOpts = append(serverOpts,
		grpc.MaxRecvMsgSize(16<<20),
		grpc.MaxSendMsgSize(16<<20),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      2 * time.Hour,
			MaxConnectionAgeGrace: 30 * time.Second,
			Time:                  2 * time.Minute,
			Timeout:               20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime:             1 * time.Minute,
				PermitWithoutStream: true,
			}),
		grpc.ChainUnaryInterceptor(unaryInts...),
		grpc.ChainStreamInterceptor(streamInts...),
	)

	grpcServer := grpc.NewServer(serverOpts...)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Let caller register their services
	registerServices(grpcServer)

	errCh := make(chan error, 1)
	go func() {
		log.Default().Info("Starting gRPC server")
		errCh <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		log.Default().Info("Shutting down gRPC server")
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		// safety timer: hard stop if graceful takes too long
		t := time.NewTimer(3 * time.Second)
		defer t.Stop()
		select {
		case <-stopped:
			return nil
		case <-t.C:
			log.Default().Info("graceful stop timed out, forcing shutdown")
			grpcServer.Stop()
			return nil
		}
	case err = <-errCh:
		wErr := errors.Wrap(err, "failed to start gRPC server")
		return wErr
	}
}
