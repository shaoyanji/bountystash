package handlers

import (
	"html/template"
	"net/http"

	"github.com/shaoyanji/bountystash/internal/http/represent"
	"github.com/shaoyanji/bountystash/internal/views"
)

var examplesIndexTemplate = template.Must(views.Parse("examples_index.tmpl"))

// HandleExamplesIndex renders a page listing all available seeded examples.
func HandleExamplesIndex(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)

	examples := SeededExamples()

	if det.Representation != represent.RepresentationHTML {
		writeJSON(w, http.StatusOK, map[string]any{
			"examples": examples,
			"count":    len(examples),
		})
		return
	}

	data := examplesIndexData{
		NavActive: "examples",
		Examples:  examples,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := examplesIndexTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

type examplesIndexData struct {
	NavActive string
	Examples  []Example
}
