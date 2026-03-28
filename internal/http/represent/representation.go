package represent

import (
	"net/http"
	"slices"
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

type Contract string

const (
	ContractNegotiated Contract = ""
	ContractHuman      Contract = "human"
	ContractAPI        Contract = "api"
	ContractPlainText  Contract = "plain"
)

// Options control representation determination for a request.
type Options struct {
	// Contract describes the route's representation boundary.
	Contract Contract
	// Explicit format override (query param or handler-provided value).
	Explicit string
	// Default is used when no override, route contract, Accept, or UA hint
	// produces a match.
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
// 2) route contract
// 3) Accept header
// 4) User-Agent hint
// 5) opts.Default (or a route-aware default when unset)
//
// The helper is intentionally small and does not alter handlers by itself; it
// prepares a shared boundary for 0.1.5 non-browser rendering.
func Determine(r *http.Request, opts Options) Determination {
	if opts.Default == "" {
		opts.Default = defaultRepresentationForContract(opts.Contract)
	}

	allowed := allowedRepresentations(opts.Contract)

	if rep, ok := parseInternalExplicit(opts.Explicit); ok && allowsRepresentation(allowed, rep) {
		return Determination{Representation: rep, Source: "explicit"}
	}

	if rep, ok := parseQueryFormat(r.URL.Query().Get("format")); ok && allowsRepresentation(allowed, rep) {
		return Determination{Representation: rep, Source: "explicit"}
	}

	if fixed, ok := fixedRepresentationForContract(opts.Contract); ok {
		return Determination{Representation: fixed, Source: "route"}
	}

	if rep, ok := fromAcceptHeader(r.Header.Get("Accept"), allowed); ok {
		return Determination{Representation: rep, Source: "accept"}
	}

	if rep, ok := fromUserAgent(r.UserAgent(), allowed); ok {
		return Determination{Representation: rep, Source: "user-agent"}
	}

	if allowsRepresentation(allowed, opts.Default) {
		return Determination{Representation: opts.Default, Source: "default"}
	}

	return Determination{
		Representation: defaultRepresentationForContract(opts.Contract),
		Source:         "default",
	}
}

func parseInternalExplicit(raw string) (Representation, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "html":
		return RepresentationHTML, true
	case "markdown", "md":
		return RepresentationMarkdown, true
	case "text", "plaintext":
		return RepresentationPlainText, true
	case "json":
		return RepresentationJSON, true
	default:
		return "", false
	}
}

func parseQueryFormat(raw string) (Representation, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "html":
		return RepresentationHTML, true
	case "md":
		return RepresentationMarkdown, true
	case "text":
		return RepresentationPlainText, true
	default:
		return "", false
	}
}

func fromAcceptHeader(raw string, allowed []Representation) (Representation, bool) {
	if raw == "" {
		return "", false
	}

	for _, part := range strings.Split(raw, ",") {
		mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))

		var rep Representation
		switch mediaType {
		case "application/json":
			rep = RepresentationJSON
		case "text/markdown", "text/x-markdown":
			rep = RepresentationMarkdown
		case "text/plain":
			rep = RepresentationPlainText
		case "text/html", "application/xhtml+xml":
			rep = RepresentationHTML
		default:
			continue
		}

		if allowsRepresentation(allowed, rep) {
			return rep, true
		}
	}

	return "", false
}

func fromUserAgent(raw string, allowed []Representation) (Representation, bool) {
	raw = strings.ToLower(raw)
	switch {
	case strings.Contains(raw, "curl"), strings.Contains(raw, "httpie"), strings.Contains(raw, "wget"):
		if allowsRepresentation(allowed, RepresentationMarkdown) {
			return RepresentationMarkdown, true
		}
		if allowsRepresentation(allowed, RepresentationPlainText) {
			return RepresentationPlainText, true
		}
	case strings.Contains(raw, "mozilla"), strings.Contains(raw, "chrome"), strings.Contains(raw, "safari"), strings.Contains(raw, "firefox"), strings.Contains(raw, "edg/"):
		if allowsRepresentation(allowed, RepresentationHTML) {
			return RepresentationHTML, true
		}
	default:
		return "", false
	}
	return "", false
}

func allowsRepresentation(allowed []Representation, rep Representation) bool {
	return slices.Contains(allowed, rep)
}

func allowedRepresentations(contract Contract) []Representation {
	switch contract {
	case ContractHuman:
		return []Representation{
			RepresentationHTML,
			RepresentationMarkdown,
			RepresentationPlainText,
		}
	case ContractAPI:
		return []Representation{RepresentationJSON}
	case ContractPlainText:
		return []Representation{RepresentationPlainText}
	default:
		return []Representation{
			RepresentationHTML,
			RepresentationMarkdown,
			RepresentationPlainText,
			RepresentationJSON,
		}
	}
}

func fixedRepresentationForContract(contract Contract) (Representation, bool) {
	switch contract {
	case ContractAPI:
		return RepresentationJSON, true
	case ContractPlainText:
		return RepresentationPlainText, true
	default:
		return "", false
	}
}

func defaultRepresentationForContract(contract Contract) Representation {
	switch contract {
	case ContractHuman:
		return RepresentationMarkdown
	case ContractAPI:
		return RepresentationJSON
	case ContractPlainText:
		return RepresentationPlainText
	default:
		return RepresentationHTML
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
