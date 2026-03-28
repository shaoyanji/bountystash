package represent

import (
	"net/http"
	"strings"
)

// Representation describes the intended human-facing response shape.
// Current surfaces primarily render HTML and JSON; Markdown and plain text
// are prepared for upcoming terminal-friendly output.
type Representation string

const (
	RepresentationHTML      Representation = "html"
	RepresentationMarkdown  Representation = "markdown"
	RepresentationPlainText Representation = "plaintext"
	RepresentationJSON      Representation = "json"
)

// Options control representation determination for a request.
type Options struct {
	// Explicit format override (query param or handler-provided value).
	Explicit string
	// Default is used when no override, Accept, or UA hint produces a match.
	Default Representation
}

// Determination captures the resolved representation and the winning signal.
type Determination struct {
	Representation Representation
	Source         string
}

// Determine resolves the best-effort representation for the request using the
// following precedence:
// 1) explicit override (opts.Explicit)
// 2) Accept header
// 3) User-Agent hint (terminal-friendly tools)
// 4) opts.Default (or HTML when unset)
//
// The helper is intentionally small and does not alter handlers by itself; it
// prepares a shared boundary for 0.1.5 non-browser rendering.
func Determine(r *http.Request, opts Options) Determination {
	if opts.Default == "" {
		opts.Default = RepresentationHTML
	}

	if rep, ok := parseExplicit(opts.Explicit); ok {
		return Determination{Representation: rep, Source: "explicit"}
	}

	if rep, ok := parseExplicit(r.URL.Query().Get("as")); ok {
		return Determination{Representation: rep, Source: "explicit"}
	}
	if rep, ok := parseExplicit(r.URL.Query().Get("format")); ok {
		return Determination{Representation: rep, Source: "explicit"}
	}

	if rep, ok := fromAcceptHeader(r.Header.Get("Accept")); ok {
		return Determination{Representation: rep, Source: "accept"}
	}

	if rep, ok := fromUserAgent(r.UserAgent()); ok {
		return Determination{Representation: rep, Source: "user-agent"}
	}

	return Determination{Representation: opts.Default, Source: "default"}
}

func parseExplicit(raw string) (Representation, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "html", "htm":
		return RepresentationHTML, true
	case "markdown", "md", "mkdown":
		return RepresentationMarkdown, true
	case "text", "txt", "plain", "plaintext":
		return RepresentationPlainText, true
	case "json":
		return RepresentationJSON, true
	default:
		return "", false
	}
}

func fromAcceptHeader(raw string) (Representation, bool) {
	if raw == "" {
		return "", false
	}

	raw = strings.ToLower(raw)

	switch {
	case strings.Contains(raw, "application/json"):
		return RepresentationJSON, true
	case strings.Contains(raw, "text/markdown") || strings.Contains(raw, "text/x-markdown"):
		return RepresentationMarkdown, true
	case strings.Contains(raw, "text/plain"):
		return RepresentationPlainText, true
	case strings.Contains(raw, "text/html"):
		return RepresentationHTML, true
	default:
		return "", false
	}
}

func fromUserAgent(raw string) (Representation, bool) {
	raw = strings.ToLower(raw)
	switch {
	case strings.Contains(raw, "curl"), strings.Contains(raw, "httpie"), strings.Contains(raw, "wget"):
		return RepresentationPlainText, true
	default:
		return "", false
	}
}

// ContentType returns the canonical response Content-Type header for the
// representation. Handlers may still choose more specific variants.
func (r Representation) ContentType() string {
	switch r {
	case RepresentationJSON:
		return "application/json; charset=utf-8"
	case RepresentationMarkdown:
		return "text/markdown; charset=utf-8"
	case RepresentationPlainText:
		return "text/plain; charset=utf-8"
	default:
		return "text/html; charset=utf-8"
	}
}
