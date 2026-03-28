package handlers

import (
	"net/http"

	"github.com/shaoyanji/bountystash/internal/http/represent"
)

// HandleManifest renders the canonical static discovery surface for curl and agents.
func HandleManifest(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)
	if det.Representation == represent.RepresentationHTML {
		det.Representation = represent.RepresentationMarkdown
	}

	writeHumanDocument(w, http.StatusOK, det.Representation, manifestDocument())
}
