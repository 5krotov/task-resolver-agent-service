package grpc

import (
	"agent-service/internal/config"
	"agent-service/internal/service/agent_service"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"log"
	"net"
	"os"

	pb "github.com/5krotov/task-resolver-pkg/grpc-api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	config  config.GRPCConfig
	server  *grpc.Server
	service *agent_service.AgentServiceServer
}

func NewServer(cfg config.GRPCConfig, service *agent_service.AgentServiceServer) (*Server, error) {
	log.Printf("running grpc server on %v ...\n", cfg.Addr)

	var server *grpc.Server
	if cfg.UseTLS {
		serverCert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			log.Fatalf("Failed to load server cert: %v", err)
		}

		caCert, err := os.ReadFile(cfg.Ca)
		if err != nil {
			log.Fatalf("Failed to read CA cert: %v", err)
		}

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caCert)

		creds := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    certPool,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to load server creds: %v", err)
		}
		server = grpc.NewServer(
			grpc.Creds(creds),
		)
	} else {
		server = grpc.NewServer()
	}
	reflection.Register(server)

	return &Server{config: cfg, server: server, service: service}, nil
}

func (s *Server) Serve() error {
	pb.RegisterAgentServiceServer(s.server, s.service)
	lis, err := net.Listen(s.config.Network, s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	log.Printf("serving grpc at %v %v", s.config.Network, s.config.Addr)
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

func (s *Server) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}
