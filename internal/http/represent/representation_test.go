package represent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeterminePrecedenceExplicitWins(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?as=markdown", nil)
	req.Header.Set("Accept", "application/json")

	det := Determine(req, Options{Default: RepresentationHTML})

	if det.Representation != RepresentationMarkdown {
		t.Fatalf("representation = %s, want markdown", det.Representation)
	}
	if det.Source != "explicit" {
		t.Fatalf("source = %s, want explicit", det.Source)
	}
}

func TestDetermineExplicitOptionBeatsQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?as=markdown", nil)

	det := Determine(req, Options{Explicit: "json", Default: RepresentationHTML})

	if det.Representation != RepresentationJSON {
		t.Fatalf("representation = %s, want json", det.Representation)
	}
	if det.Source != "explicit" {
		t.Fatalf("source = %s, want explicit", det.Source)
	}
}

func TestDetermineAcceptFallsBack(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/markdown, text/html")

	det := Determine(req, Options{Default: RepresentationHTML})

	if det.Representation != RepresentationMarkdown {
		t.Fatalf("representation = %s, want markdown", det.Representation)
	}
	if det.Source != "accept" {
		t.Fatalf("source = %s, want accept", det.Source)
	}
}

func TestDetermineAcceptJSONPreferred(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json, text/html")

	det := Determine(req, Options{Default: RepresentationHTML})

	if det.Representation != RepresentationJSON {
		t.Fatalf("representation = %s, want json", det.Representation)
	}
	if det.Source != "accept" {
		t.Fatalf("source = %s, want accept", det.Source)
	}
}

func TestDetermineUserAgentHintOnlyWhenNoOtherSignals(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")

	det := Determine(req, Options{Default: RepresentationHTML})

	if det.Representation != RepresentationPlainText {
		t.Fatalf("representation = %s, want plaintext", det.Representation)
	}
	if det.Source != "user-agent" {
		t.Fatalf("source = %s, want user-agent", det.Source)
	}
}

func TestDetermineDefaultBrowserStaysHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	det := Determine(req, Options{Default: RepresentationHTML})

	if det.Representation != RepresentationHTML {
		t.Fatalf("representation = %s, want html", det.Representation)
	}
	if det.Source != "accept" {
		t.Fatalf("source = %s, want accept", det.Source)
	}
}

func TestDetermineFallsBackToDefaultWhenNoSignals(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	det := Determine(req, Options{Default: RepresentationPlainText})

	if det.Representation != RepresentationPlainText {
		t.Fatalf("representation = %s, want plaintext", det.Representation)
	}
	if det.Source != "default" {
		t.Fatalf("source = %s, want default", det.Source)
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
