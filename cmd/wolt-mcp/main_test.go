package main

import (
	"testing"
	"time"
)

func TestResolveWoltRequestMinInterval_Default(t *testing.T) {
	t.Setenv(woltHTTPMinIntervalEnv, "")
	if got := resolveWoltRequestMinInterval(); got != defaultWoltHTTPMinInterval {
		t.Errorf("resolveWoltRequestMinInterval() = %v, want %v", got, defaultWoltHTTPMinInterval)
	}
}

func TestResolveWoltRequestMinInterval_ValidValue(t *testing.T) {
	t.Setenv(woltHTTPMinIntervalEnv, "500")
	if got, want := resolveWoltRequestMinInterval(), 500*time.Millisecond; got != want {
		t.Errorf("resolveWoltRequestMinInterval() = %v, want %v", got, want)
	}
}

func TestResolveWoltRequestMinInterval_Zero(t *testing.T) {
	t.Setenv(woltHTTPMinIntervalEnv, "0")
	if got := resolveWoltRequestMinInterval(); got != 0 {
		t.Errorf("resolveWoltRequestMinInterval() = %v, want 0", got)
	}
}

func TestResolveWoltRequestMinInterval_NegativeFallsBackToDefault(t *testing.T) {
	t.Setenv(woltHTTPMinIntervalEnv, "-10")
	if got := resolveWoltRequestMinInterval(); got != defaultWoltHTTPMinInterval {
		t.Errorf("resolveWoltRequestMinInterval() = %v, want %v (default for negative)", got, defaultWoltHTTPMinInterval)
	}
}

func TestResolveWoltRequestMinInterval_NonNumericFallsBackToDefault(t *testing.T) {
	t.Setenv(woltHTTPMinIntervalEnv, "fast")
	if got := resolveWoltRequestMinInterval(); got != defaultWoltHTTPMinInterval {
		t.Errorf("resolveWoltRequestMinInterval() = %v, want %v (default for non-numeric)", got, defaultWoltHTTPMinInterval)
	}
}
