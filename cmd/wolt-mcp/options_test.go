package main

import (
	"strings"
	"testing"
)

func TestParseOptions_DefaultStdioAndLocale(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.listen != "" {
		t.Errorf("listen = %q, want empty (stdio)", opt.listen)
	}
	if opt.locale != defaultLocale {
		t.Errorf("locale = %q, want %q", opt.locale, defaultLocale)
	}
}

func TestParseOptions_LocaleEnvFallback(t *testing.T) {
	t.Setenv(localeEnv, "  fi-FI  ")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.locale != "fi-FI" {
		t.Errorf("locale = %q, want %q", opt.locale, "fi-FI")
	}
}

func TestParseOptions_LocaleFlagOverridesEnv(t *testing.T) {
	t.Setenv(localeEnv, "fi-FI")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--locale", "de-DE"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.locale != "de-DE" {
		t.Errorf("locale = %q, want %q", opt.locale, "de-DE")
	}
}

func TestParseOptions_LocaleEqualsForm(t *testing.T) {
	t.Setenv(localeEnv, "sv-SE")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--locale=de-DE"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.locale != "de-DE" {
		t.Errorf("locale = %q, want %q", opt.locale, "de-DE")
	}
}

func TestParseOptions_LocaleFlagTrimsWhitespace(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--locale", "  pl-PL  "})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.locale != "pl-PL" {
		t.Errorf("locale = %q, want %q", opt.locale, "pl-PL")
	}
}

func TestParseOptions_EmptyLocaleFlagFallsThroughToEnv(t *testing.T) {
	t.Setenv(localeEnv, "et-EE")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--locale="})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.locale != "et-EE" {
		t.Errorf("locale = %q, want env fallback %q", opt.locale, "et-EE")
	}
}

func TestParseOptions_LocaleLastFlagWins(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--locale=first", "--locale", "second"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.locale != "second" {
		t.Errorf("locale = %q, want last flag %q", opt.locale, "second")
	}
}

func TestParseOptions_UnknownFlagErrors(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	_, err := parseOptions([]string{"--something", "value"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestParseOptions_VersionAndHelp(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	for _, args := range [][]string{
		{"--version"},
		{"-v"},
		{"version"},
	} {
		opt, err := parseOptions(args)
		if err != nil {
			t.Fatalf("parseOptions(%q): %v", args, err)
		}
		if !opt.version {
			t.Errorf("parseOptions(%q).version = false", args)
		}
	}

	for _, args := range [][]string{
		{"--help"},
		{"-h"},
		{"help"},
	} {
		opt, err := parseOptions(args)
		if err != nil {
			t.Fatalf("parseOptions(%q): %v", args, err)
		}
		if !opt.help {
			t.Errorf("parseOptions(%q).help = false", args)
		}
	}
}

func TestParseOptions_ListenPortOnlyBindsLoopback(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--listen", "8080"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.listen != "127.0.0.1:8080" {
		t.Errorf("listen = %q, want 127.0.0.1:8080", opt.listen)
	}
	if opt.listenURL() != "http://127.0.0.1:8080/mcp" {
		t.Errorf("listenURL = %q", opt.listenURL())
	}
}

func TestParseOptions_ListenEnv(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "127.0.0.1:9090")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.listen != "127.0.0.1:9090" {
		t.Errorf("listen = %q, want 127.0.0.1:9090", opt.listen)
	}
}

func TestParseOptions_NonLoopbackRequiresToken(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	for _, listen := range []string{":8080", "0.0.0.0:8080", "192.168.1.9:8080"} {
		_, err := parseOptions([]string{"--listen", listen})
		if err == nil {
			t.Errorf("listen %q: expected token required error", listen)
			continue
		}
		if !strings.Contains(err.Error(), "requires --token") {
			t.Errorf("listen %q: error %q, want requires --token", listen, err)
		}
	}
}

func TestParseOptions_NonLoopbackWithTokenOK(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--listen", "0.0.0.0:8080", "--token", "secret"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.listen != "0.0.0.0:8080" {
		t.Errorf("listen = %q, want 0.0.0.0:8080", opt.listen)
	}
	if opt.token != "secret" {
		t.Errorf("token = %q, want secret", opt.token)
	}
}

func TestParseOptions_TokenFromEnvAllowsNonLoopback(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "from-env")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--listen", ":8080"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.token != "from-env" {
		t.Errorf("token = %q, want from-env", opt.token)
	}
	if opt.listen != ":8080" {
		t.Errorf("listen = %q, want :8080", opt.listen)
	}
	if opt.listenURL() != "http://0.0.0.0:8080/mcp" {
		t.Errorf("listenURL = %q", opt.listenURL())
	}
}

func TestParseOptions_TLSRequiresBothFiles(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	_, err := parseOptions([]string{"--listen", "127.0.0.1:8443", "--tls-cert", "cert.pem"})
	if err == nil || !strings.Contains(err.Error(), "both be set") {
		t.Fatalf("error = %v, want both be set", err)
	}

	opt, err := parseOptions([]string{
		"--listen", "127.0.0.1:8443",
		"--tls-cert", "cert.pem",
		"--tls-key", "key.pem",
	})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.httpScheme() != "https" {
		t.Errorf("scheme = %q, want https", opt.httpScheme())
	}
	if opt.listenURL() != "https://127.0.0.1:8443/mcp" {
		t.Errorf("listenURL = %q", opt.listenURL())
	}
}

func TestParseOptions_HTTPFlagsRequireListen(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	_, err := parseOptions([]string{"--token", "secret"})
	if err == nil || !strings.Contains(err.Error(), "require --listen") {
		t.Fatalf("error = %v, want require --listen", err)
	}
}

func TestParseOptions_EnvTokenWithoutListenStaysStdio(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "from-env")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.listen != "" {
		t.Errorf("listen = %q, want empty stdio", opt.listen)
	}
}

func TestParseOptions_LocalhostIsLoopback(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--listen", "localhost:8080"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.listen != "localhost:8080" {
		t.Errorf("listen = %q, want localhost:8080", opt.listen)
	}
}

func TestParseOptions_IPv6Loopback(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	opt, err := parseOptions([]string{"--listen", "[::1]:8080"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if opt.listen != "[::1]:8080" {
		t.Errorf("listen = %q, want [::1]:8080", opt.listen)
	}
}

func TestParseOptions_InvalidListen(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	_, err := parseOptions([]string{"--listen", "not-a-port"})
	if err == nil {
		t.Fatal("expected invalid listen error")
	}
}

func TestParseOptions_UnexpectedArgument(t *testing.T) {
	t.Setenv(localeEnv, "")
	t.Setenv(listenEnv, "")
	t.Setenv(tokenEnv, "")
	t.Setenv(tlsCertEnv, "")
	t.Setenv(tlsKeyEnv, "")

	_, err := parseOptions([]string{"stdio"})
	if err == nil || !strings.Contains(err.Error(), "unexpected argument") {
		t.Fatalf("error = %v, want unexpected argument", err)
	}
}
