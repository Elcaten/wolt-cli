package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

const (
	listenEnv  = "WOLT_MCP_LISTEN"
	tokenEnv   = "WOLT_MCP_TOKEN"
	tlsCertEnv = "WOLT_MCP_TLS_CERT"
	tlsKeyEnv  = "WOLT_MCP_TLS_KEY"

	mcpHTTPPath   = "/mcp"
	mcpHealthPath = "/healthz"
)

// options is the parsed wolt-mcp command line. An empty Listen means stdio.
type options struct {
	locale  string
	listen  string
	tlsCert string
	tlsKey  string
	token   string
	version bool
	help    bool
}

func parseOptions(args []string) (options, error) {
	if len(args) == 1 {
		switch args[0] {
		case "version":
			return options{version: true}, nil
		case "help":
			return options{help: true}, nil
		}
	}

	var opt options
	fs := pflag.NewFlagSet("wolt-mcp", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opt.locale, "locale", "", "response locale in BCP-47 format")
	fs.StringVar(&opt.listen, "listen", "", "host:port or port to serve Streamable HTTP (default stdio)")
	fs.StringVar(&opt.tlsCert, "tls-cert", "", "TLS certificate file (requires --tls-key)")
	fs.StringVar(&opt.tlsKey, "tls-key", "", "TLS private key file (requires --tls-cert)")
	fs.StringVar(&opt.token, "token", "", "bearer token required by HTTP clients")
	fs.BoolVarP(&opt.version, "version", "v", false, "print version and exit")
	fs.BoolVarP(&opt.help, "help", "h", false, "print help and exit")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if leftover := fs.Args(); len(leftover) > 0 {
		return options{}, fmt.Errorf("unexpected argument %q", leftover[0])
	}

	opt.locale = strings.TrimSpace(opt.locale)
	opt.listen = strings.TrimSpace(opt.listen)
	opt.tlsCert = strings.TrimSpace(opt.tlsCert)
	opt.tlsKey = strings.TrimSpace(opt.tlsKey)
	opt.token = strings.TrimSpace(opt.token)

	if opt.locale == "" {
		opt.locale = firstNonEmpty(strings.TrimSpace(os.Getenv(localeEnv)), defaultLocale)
	}

	if opt.version || opt.help {
		return opt, nil
	}

	if opt.listen == "" {
		opt.listen = strings.TrimSpace(os.Getenv(listenEnv))
	}
	if opt.listen == "" {
		if opt.token != "" || opt.tlsCert != "" || opt.tlsKey != "" {
			return options{}, fmt.Errorf("--tls-cert, --tls-key, and --token require --listen")
		}
		return opt, nil
	}

	if opt.tlsCert == "" {
		opt.tlsCert = strings.TrimSpace(os.Getenv(tlsCertEnv))
	}
	if opt.tlsKey == "" {
		opt.tlsKey = strings.TrimSpace(os.Getenv(tlsKeyEnv))
	}
	if opt.token == "" {
		opt.token = strings.TrimSpace(os.Getenv(tokenEnv))
	}
	if err := opt.normalizeAndValidate(); err != nil {
		return options{}, err
	}
	return opt, nil
}

func (o *options) normalizeAndValidate() error {
	addr, err := normalizeListenAddr(o.listen)
	if err != nil {
		return err
	}
	o.listen = addr

	if (o.tlsCert == "") != (o.tlsKey == "") {
		return fmt.Errorf("--tls-cert and --tls-key must both be set")
	}

	loopback, err := isLoopbackListenAddr(o.listen)
	if err != nil {
		return err
	}
	if !loopback && o.token == "" {
		return fmt.Errorf("non-loopback --listen %s requires --token (or %s)", o.listen, tokenEnv)
	}
	return nil
}

func (o options) httpScheme() string {
	if o.tlsCert != "" {
		return "https"
	}
	return "http"
}

func (o options) listenURL() string {
	host := o.listen
	if strings.HasPrefix(host, ":") {
		host = "0.0.0.0" + host
	}
	return o.httpScheme() + "://" + host + mcpHTTPPath
}

func normalizeListenAddr(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty --listen address")
	}
	if !strings.Contains(raw, ":") {
		if _, err := strconv.Atoi(raw); err != nil {
			return "", fmt.Errorf("invalid --listen %q: want host:port or port", raw)
		}
		return net.JoinHostPort("127.0.0.1", raw), nil
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return "", fmt.Errorf("invalid --listen %q: %w", raw, err)
	}
	if port == "" {
		return "", fmt.Errorf("invalid --listen %q: missing port", raw)
	}
	return net.JoinHostPort(host, port), nil
}

func isLoopbackListenAddr(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, fmt.Errorf("invalid --listen %q: %w", addr, err)
	}
	if host == "" {
		return false, nil
	}
	if strings.EqualFold(host, "localhost") {
		return true, nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false, nil
	}
	return ip.IsLoopback(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
