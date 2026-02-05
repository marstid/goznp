package daemon

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/marstid/goznp/internal/server"
	"github.com/marstid/goznp/pkg/adapter"
	"github.com/marstid/goznp/pkg/event"
	"github.com/marstid/goznp/pkg/message"
	"github.com/marstid/goznp/pkg/state"
)

// Server manages the HTTP server and associated components.
type Server struct {
	config     *Config
	adapter    *adapter.Adapter
	manager    *state.Manager
	eventBus   *event.Bus
	listener   *message.Listener
	httpServer *http.Server
	logger     *slog.Logger
}

// NewServer creates a new daemon server.
func NewServer(cfg *Config, adapter *adapter.Adapter, manager *state.Manager, eventBus *event.Bus, listener *message.Listener, logger *slog.Logger) *Server {
	return &Server{
		config:   cfg,
		adapter:  adapter,
		manager:  manager,
		eventBus: eventBus,
		listener: listener,
		logger:   logger,
	}
}

// Start starts the HTTP server.
// This blocks until the server is stopped.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Apply middleware
	var handler http.Handler = mux
	handler = server.LoggingMiddleware(s.logger)(handler)
	handler = server.RecoverMiddleware(s.logger)(handler)
	handler = server.CORSMiddleware(handler)

	// Setup routes
	server.SetupRoutes(mux, s.manager, s.adapter, s.logger)

	s.httpServer = &http.Server{
		Addr:         s.config.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.Info("HTTP server starting", "addr", s.config.HTTPAddr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("HTTP server shutting down")
	return s.httpServer.Shutdown(ctx)
}

// GetAddr returns the server's listen address.
func (s *Server) GetAddr() string {
	return s.config.HTTPAddr
}
