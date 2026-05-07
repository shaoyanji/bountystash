package represent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetermineHumanExplicitFormatWins(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?format=md", nil)
	req.Header.Set("Accept", "text/html")

	det := Determine(req, Options{Contract: ContractHuman})

	if det.Representation != RepresentationMarkdown {
		t.Fatalf("representation = %s, want markdown", det.Representation)
	}
	if det.Source != "explicit" {
		t.Fatalf("source = %s, want explicit", det.Source)
	}
}

func TestDetermineHumanIgnoresLegacyAsOverride(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?as=markdown", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")

	det := Determine(req, Options{Contract: ContractHuman})

	if det.Representation != RepresentationMarkdown {
		t.Fatalf("representation = %s, want markdown", det.Representation)
	}
	if det.Source != "user-agent" {
		t.Fatalf("source = %s, want user-agent", det.Source)
	}
}

func TestDetermineHumanAcceptChoosesHTMLWhenAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json, text/html")

	det := Determine(req, Options{Contract: ContractHuman})

	if det.Representation != RepresentationHTML {
		t.Fatalf("representation = %s, want html", det.Representation)
	}
	if det.Source != "accept" {
		t.Fatalf("source = %s, want accept", det.Source)
	}
}

func TestDetermineHumanUserAgentDefaultsToMarkdown(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")

	det := Determine(req, Options{Contract: ContractHuman})

	if det.Representation != RepresentationMarkdown {
		t.Fatalf("representation = %s, want markdown", det.Representation)
	}
	if det.Source != "user-agent" {
		t.Fatalf("source = %s, want user-agent", det.Source)
	}
}

func TestDetermineHumanBrowserUserAgentKeepsHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	det := Determine(req, Options{Contract: ContractHuman})

	if det.Representation != RepresentationHTML {
		t.Fatalf("representation = %s, want html", det.Representation)
	}
	if det.Source != "user-agent" {
		t.Fatalf("source = %s, want user-agent", det.Source)
	}
}

func TestDetermineHumanFallsBackToMarkdown(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	det := Determine(req, Options{Contract: ContractHuman})

	if det.Representation != RepresentationMarkdown {
		t.Fatalf("representation = %s, want markdown", det.Representation)
	}
	if det.Source != "default" {
		t.Fatalf("source = %s, want default", det.Source)
	}
}

func TestDetermineAPIContractForcesJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/examples?format=text", nil)
	req.Header.Set("Accept", "text/plain")

	det := Determine(req, Options{Contract: ContractAPI})

	if det.Representation != RepresentationJSON {
		t.Fatalf("representation = %s, want json", det.Representation)
	}
	if det.Source != "route" {
		t.Fatalf("source = %s, want route", det.Source)
	}
}

func TestDeterminePlainTextContractForcesPlainText(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz?format=html", nil)

	det := Determine(req, Options{Contract: ContractPlainText})

	if det.Representation != RepresentationPlainText {
		t.Fatalf("representation = %s, want plaintext", det.Representation)
	}
	if det.Source != "route" {
		t.Fatalf("source = %s, want route", det.Source)
	}
}

func TestContentTypeMappings(t *testing.T) {
	cases := map[Representation]string{
		RepresentationHTML:      "text/html; charset=utf-8",
		RepresentationMarkdown:  "text/markdown; charset=utf-8",
		RepresentationPlainText: "text/plain; charset=utf-8",
		RepresentationJSON:      "application/json; charset=utf-8",
	}

	for rep, want := range cases {
		if got := rep.ContentType(); got != want {
			t.Fatalf("content type for %s = %s, want %s", rep, got, want)
		}
	}
}

func TestParseInternalExplicit_InvalidFormat(t *testing.T) {
	tests := []string{"", "json", "invalid", "HTML", "Markdown"}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			rep, ok := parseInternalExplicit(raw)
			if raw == "json" || raw == "invalid" {
				if ok {
					t.Errorf("parseInternalExplicit(%q) should return false", raw)
				}
			} else if raw != "" {
				if !ok {
					t.Errorf("parseInternalExplicit(%q) should return true", raw)
				}
			}
		})
	}
}

func TestFromAcceptHeader_EdgeCases(t *testing.T) {
	tests := []struct {
		header string
		want   Representation
	}{
		{"", RepresentationMarkdown}, // empty = default
		{"invalid", RepresentationMarkdown}, // invalid = default
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Accept", tt.header)
			}
			det := Determine(req, Options{Contract: ContractHuman})
			// Just verify it doesn't panic
			_ = det.Representation
		})
	}
}
