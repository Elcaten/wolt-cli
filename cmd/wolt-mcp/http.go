package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultHTTPSessionTimeout = 30 * time.Minute
	httpReadHeaderTimeout     = 10 * time.Second
	httpShutdownTimeout       = 5 * time.Second
)

func newMCPHTTPHandler(srv *mcp.Server, token string, logger *slog.Logger) http.Handler {
	stream := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, &mcp.StreamableHTTPOptions{
		Logger:         logger,
		SessionTimeout: defaultHTTPSessionTimeout,
	})
	var handler http.Handler = stream
	if token != "" {
		handler = auth.RequireBearerToken(staticTokenVerifier(token), &auth.RequireBearerTokenOptions{
			AllowMissingExpiration: true,
		})(stream)
	}
	mux := http.NewServeMux()
	mux.Handle(mcpHTTPPath, handler)
	mux.HandleFunc(mcpHealthPath, healthz)
	return mux
}

func healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte("ok\n"))
}

func runHTTP(ctx context.Context, logger *slog.Logger, srv *mcp.Server, opt options) error {
	httpSrv := &http.Server{
		Addr:              opt.listen,
		Handler:           newMCPHTTPHandler(srv, opt.token, logger),
		ReadHeaderTimeout: httpReadHeaderTimeout,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	ln, err := net.Listen("tcp", opt.listen)
	if err != nil {
		return fmt.Errorf("listen %s: %w", opt.listen, err)
	}

	errCh := make(chan error, 1)
	go func() {
		var serveErr error
		if opt.tlsCert != "" {
			serveErr = httpSrv.ServeTLS(ln, opt.tlsCert, opt.tlsKey)
		} else {
			serveErr = httpSrv.Serve(ln)
		}
		_ = ln.Close()
		errCh <- serveErr
	}()

	logger.Info("wolt-mcp listening", "url", opt.listenURL())

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		err := <-errCh
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func staticTokenVerifier(want string) auth.TokenVerifier {
	wantBytes := []byte(want)
	return func(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		if subtle.ConstantTimeCompare([]byte(token), wantBytes) != 1 {
			return nil, fmt.Errorf("%w: token mismatch", auth.ErrInvalidToken)
		}
		return &auth.TokenInfo{UserID: "wolt-mcp"}, nil
	}
}
