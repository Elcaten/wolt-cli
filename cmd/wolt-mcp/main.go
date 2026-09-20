// Command wolt-mcp serves wolt-cli's functionality over the Model Context
// Protocol so AI clients (Claude Desktop, Claude Code, Cursor, …) can drive
// Wolt searches, view orders, manage baskets, and preview checkouts.
//
// Wire it into Claude Desktop / Claude Code with:
//
//	{ "mcpServers": { "wolt": { "command": "wolt-mcp" } } }
//
// Or serve Streamable HTTP(S):
//
//	wolt-mcp --listen 127.0.0.1:8080
//	{ "mcpServers": { "wolt": { "url": "http://127.0.0.1:8080/mcp" } } }
//
// The server shares ~/.wolt/.wolt-config.json with the wolt CLI binary — log
// in once via `wolt login` and the MCP server inherits the same session.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mekedron/wolt-cli/internal/config"
	locationgateway "github.com/mekedron/wolt-cli/internal/gateway/location"
	woltgateway "github.com/mekedron/wolt-cli/internal/gateway/wolt"
	"github.com/mekedron/wolt-cli/internal/mcpserver"
	"github.com/mekedron/wolt-cli/internal/service/profile"
)

var version = "dev"

const (
	defaultWoltHTTPMinInterval = 220 * time.Millisecond
	woltHTTPMinIntervalEnv     = "WOLT_HTTP_MIN_INTERVAL_MS"
	defaultLocale              = "en-FI"
	localeEnv                  = "WOLT_LOCALE"
	duplicateContentEnv        = "WOLT_MCP_DUPLICATE_CONTENT"
)

func main() {
	// CRITICAL: stdout is the MCP JSON-RPC transport in stdio mode. Anything
	// that lands on stdout outside of the SDK will corrupt the protocol. Force
	// the stdlib `log` package and the default slog handler to stderr before
	// any other init can fire a log line.
	log.SetOutput(os.Stderr)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	opt, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "wolt-mcp: %v\n", err)
		os.Exit(2)
	}
	if opt.version {
		fmt.Println(version)
		return
	}
	if opt.help {
		printHelp()
		return
	}

	store, err := config.NewStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wolt-mcp:", err)
		os.Exit(1)
	}

	wolt := woltgateway.NewClient(
		woltgateway.WithRequestMinInterval(resolveWoltRequestMinInterval()),
		woltgateway.WithLocale(opt.locale),
	)

	deps := mcpserver.Deps{
		Wolt:             wolt,
		Profiles:         profile.NewResolver(store),
		Location:         locationgateway.NewClient(),
		Config:           store,
		Version:          version,
		Locale:           opt.locale,
		Logger:           logger,
		DuplicateContent: resolveDuplicateContent(),
	}

	srv := mcpserver.NewServer(deps)

	if opt.listen == "" {
		logger.Info("wolt-mcp starting", "version", version, "config", store.Path(), "transport", "stdio")
		if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			logger.Error("wolt-mcp exited with error", "err", err)
			os.Exit(1)
		}
		return
	}

	logger.Info("wolt-mcp starting", "version", version, "config", store.Path(), "transport", "http", "listen", opt.listen)
	if err := serveHTTP(logger, srv, opt); err != nil {
		logger.Error("wolt-mcp exited with error", "err", err)
		os.Exit(1)
	}
}

func serveHTTP(logger *slog.Logger, srv *mcp.Server, opt options) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runHTTP(ctx, logger, srv, opt)
}

func printHelp() {
	fmt.Println(strings.TrimSpace(`
wolt-mcp — Model Context Protocol server for wolt-cli.

Usage:
  wolt-mcp                         Run the MCP server over stdio.
  wolt-mcp --listen [host:]port    Serve Streamable HTTP at /mcp.
  wolt-mcp --version               Print version and exit.
  wolt-mcp --help                  Print this message and exit.

Options:
  --locale <bcp47>      Response locale in BCP-47 format (default: en-FI).
                        Can also be set via the WOLT_LOCALE environment variable.
  --listen <addr>       Serve Streamable HTTP on host:port, or a port (binds
                        127.0.0.1). Default is stdio. Also WOLT_MCP_LISTEN.
  --tls-cert <file>     TLS certificate (requires --tls-key). WOLT_MCP_TLS_CERT.
  --tls-key <file>      TLS private key (requires --tls-cert). WOLT_MCP_TLS_KEY.
  --token <secret>      Require Authorization: Bearer for HTTP clients.
                        Required when --listen is not loopback. WOLT_MCP_TOKEN.

Environment:
  WOLT_MCP_DUPLICATE_CONTENT=1
                        Also serve the full typed payload as serialized JSON in
                        content, not just structuredContent. Set this only for
                        clients that read content alone — it roughly doubles
                        response size.

Wire into an MCP client over stdio (Claude Desktop, Claude Code, Cursor):

  { "mcpServers": { "wolt": { "command": "wolt-mcp" } } }

Or over Streamable HTTP:

  wolt-mcp --listen 127.0.0.1:8080
  { "mcpServers": { "wolt": { "url": "http://127.0.0.1:8080/mcp" } } }

HTTP exposes the same Wolt login session as the CLI. Bind loopback unless you
set --token, and use --tls-cert/--tls-key for HTTPS.

Authentication is shared with the wolt CLI — run 'wolt login' once to enable
the auth-gated tools (cart, favorites, account, checkout_preview).
`))
}

func resolveWoltRequestMinInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(woltHTTPMinIntervalEnv))
	if raw == "" {
		return defaultWoltHTTPMinInterval
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return defaultWoltHTTPMinInterval
	}
	return time.Duration(ms) * time.Millisecond
}

// resolveDuplicateContent reads the opt-in that mirrors the full typed payload
// into Content. Only the documented truthy spellings enable it, so a stray or
// empty value keeps the compact default rather than silently doubling payloads.
func resolveDuplicateContent() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(duplicateContentEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
