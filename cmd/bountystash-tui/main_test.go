package main

import (
	"errors"
	"strings"
	"testing"

	clientapi "github.com/shaoyanji/bountystash/internal/client/api"
	"github.com/shaoyanji/bountystash/internal/http/handlers"
	"github.com/shaoyanji/bountystash/internal/packets"
)

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

func TestFormatValidationErrorsSorted(t *testing.T) {
	out := formatValidationErrors(packets.ValidationErrors{
		"title": "required",
		"kind":  "invalid",
	})
	if out != "validation failed: kind: invalid | title: required" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestDescribeErrorAPIValidation(t *testing.T) {
	out := describeError(&clientapi.APIError{
		StatusCode: 400,
		Message:    "validation failed",
		ValidationErrors: packets.ValidationErrors{
			"title": "required",
		},
	})
	if !strings.Contains(out, "validation failed: title: required") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestDescribeErrorInvalidJSON(t *testing.T) {
	out := describeError(errors.New("decode GET /api/work: invalid JSON response: invalid character"))
	if out != "invalid response from backend" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFormatWorkDetailIncludesCoreFields(t *testing.T) {
	out := formatWorkDetail(handlers.WorkDetail{
		ID:           "w1",
		Status:       "open",
		Version:      1,
		ExactHash:    "abc",
		QuotientHash: "def",
		Packet: packets.NormalizedPacket{
			Title:              "T",
			Kind:               packets.KindBounty,
			Visibility:         packets.VisibilityPublic,
			RewardModel:        "fixed",
			Scope:              []string{"s1"},
			Deliverables:       []string{"d1"},
			AcceptanceCriteria: []string{"a1"},
		},
	})
	for _, want := range []string{
		"Work Item ID: w1",
		"Status: open",
		"Exact Hash: abc",
		"Quotient Hash: def",
		"Scope",
		"Deliverables",
		"Acceptance Criteria",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %q", want, out)
		}
	}
}
