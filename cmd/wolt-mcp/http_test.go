package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mekedron/wolt-cli/internal/mcpserver"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testMCPServer() *mcp.Server {
	return mcpserver.NewServer(mcpserver.Deps{Version: "v0.0.0-test"})
}

type headerRoundTripper struct {
	base   http.RoundTripper
	header http.Header
}

func (t headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	for key, values := range t.header {
		clone.Header.Del(key)
		for _, value := range values {
			clone.Header.Add(key, value)
		}
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

func TestHTTPHandler_ToolsList(t *testing.T) {
	httpServer := httptest.NewServer(newMCPHTTPHandler(testMCPServer(), "", discardLogger()))
	t.Cleanup(httpServer.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "http-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL + mcpHTTPPath,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	names := []string{}
	for tool, iterErr := range session.Tools(ctx, nil) {
		if iterErr != nil {
			t.Fatalf("listing tools: %v", iterErr)
		}
		names = append(names, tool.Name)
	}
	if len(names) < 20 {
		t.Errorf("expected at least 20 tools, got %d: %v", len(names), names)
	}
	for _, must := range []string{"wolt_feed", "wolt_cart_show", "wolt_account_status", "wolt_checkout_preview"} {
		if !slices.Contains(names, must) {
			t.Errorf("missing tool %q (got %d tools)", must, len(names))
		}
	}
}

func TestHTTPHandler_BearerRequired(t *testing.T) {
	const token = "test-token"
	httpServer := httptest.NewServer(newMCPHTTPHandler(testMCPServer(), token, discardLogger()))
	t.Cleanup(httpServer.Close)

	resp, err := http.Post(httpServer.URL+mcpHTTPPath, "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if err != nil {
		t.Fatalf("POST without token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST without token status = %d, want 401", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "http-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + mcpHTTPPath,
		HTTPClient: &http.Client{Transport: headerRoundTripper{
			header: http.Header{"Authorization": []string{"Bearer " + token}},
		}},
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("client.Connect with bearer: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	found := false
	for tool, iterErr := range session.Tools(ctx, nil) {
		if iterErr != nil {
			t.Fatalf("listing tools: %v", iterErr)
		}
		if tool.Name == "wolt_feed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("authenticated tools/list did not return wolt_feed")
	}
}

func TestHTTPHandler_RootIsNotMCP(t *testing.T) {
	httpServer := httptest.NewServer(newMCPHTTPHandler(testMCPServer(), "", discardLogger()))
	t.Cleanup(httpServer.Close)

	resp, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET / status = %d, want 404", resp.StatusCode)
	}
}

func TestHTTPHandler_WrongBearerRejected(t *testing.T) {
	httpServer := httptest.NewServer(newMCPHTTPHandler(testMCPServer(), "correct", discardLogger()))
	t.Cleanup(httpServer.Close)

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+mcpHTTPPath, strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer wrong")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong bearer status = %d, want 401", resp.StatusCode)
	}
}
