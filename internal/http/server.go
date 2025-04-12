package http

import (
	"agent-service/internal/config"
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

type Server struct {
	Config config.HTTPConfig
	Server *http.Server
	Mux    *http.ServeMux
}

func NewServer(cfg config.HTTPConfig) *Server {
	return &Server{
		Config: cfg,
		Server: nil,
		Mux:    http.NewServeMux(),
	}
}

func (s *Server) Run() {
	log.Printf("running http server on %v ...\n", s.Config.Addr)
	s.Server = &http.Server{
		Addr:    s.Config.Addr,
		Handler: s.Mux,
	}

	if err := s.Server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen and serve http server faild: %v", err)
	}
}

func (s *Server) Stop() {
	if s.Server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("shutdown http server failed: %v", err)
		}
	}
}
