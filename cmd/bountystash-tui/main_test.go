package main

import "testing"

func TestResolveBaseURLPrefersFlag(t *testing.T) {
	t.Setenv("BOUNTYSTASH_BASE_URL", "https://env.example")
	got := resolveBaseURL("https://flag.example")
	if got != "https://flag.example" {
		t.Fatalf("expected flag value, got %q", got)
	}
}

func TestResolveBaseURLEnvFallback(t *testing.T) {
	t.Setenv("BOUNTYSTASH_BASE_URL", "https://env.example")
	got := resolveBaseURL("")
	if got != "https://env.example" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestResolveBaseURLCompiledDefault(t *testing.T) {
	t.Setenv("BOUNTYSTASH_BASE_URL", "")
	got := resolveBaseURL("")
	if got != compiledDefaultBaseURL {
		t.Fatalf("expected compiled default %q, got %q", compiledDefaultBaseURL, got)
	}
}
