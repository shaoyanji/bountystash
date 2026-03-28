package packets

import "testing"

func TestNormalizeDraftPrivateSecurityForcesPrivateVisibility(t *testing.T) {
	in := DraftInput{
		Title:       "Sec report",
		Kind:        "private_security",
		Visibility:  "public",
		Scope:       "one\r\ntwo",
		RewardModel: "fixed",
	}

	got := NormalizeDraft(in)
	if got.Visibility != VisibilityPrivate {
		t.Fatalf("expected visibility %q, got %q", VisibilityPrivate, got.Visibility)
	}
	if len(got.Scope) != 2 || got.Scope[0] != "one" || got.Scope[1] != "two" {
		t.Fatalf("unexpected normalized scope: %#v", got.Scope)
	}
}

func TestValidateDraftInput(t *testing.T) {
	errs := ValidateDraftInput(DraftInput{
		Title:      " ",
		Kind:       "invalid",
		Visibility: "world",
	})

	if errs.Empty() {
		t.Fatal("expected validation errors")
	}
	if errs["title"] == "" || errs["kind"] == "" || errs["visibility"] == "" {
		t.Fatalf("missing expected field errors: %#v", errs)
	}
}
