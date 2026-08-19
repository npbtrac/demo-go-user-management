package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Server is an HTTP listener with graceful shutdown.
type Server struct {
	name   string
	addr   string
	server *http.Server
}

func NewServer(name, addr string, handler http.Handler) *Server {
	return &Server{
		name: name,
		addr: addr,
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Serve(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	slog.Info("http listening", "surface", s.name, "addr", ln.Addr().String())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(ln)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
