package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/shaoyanji/bountystash/internal/http/represent"
	"github.com/shaoyanji/bountystash/internal/service"
	"github.com/shaoyanji/bountystash/internal/views"
)

var searchTemplate = template.Must(views.Parse("search.tmpl"))

type SearchHandler struct {
	svc service.WorkService
}

func NewSearchHandler(svc service.WorkService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// HandleSearch handles both HTML and JSON search requests.
func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	det := humanRouteRepresentation(r)

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		if det.Representation == represent.RepresentationHTML {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		writeJSON(w, http.StatusBadRequest, apiError{Error: "missing query parameter 'q'"})
		return
	}

	limit := defaultListLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			if det.Representation == represent.RepresentationHTML {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusBadRequest, apiError{Error: "invalid limit"})
			return
		}
		if parsed <= 0 || parsed > maxListLimit {
			parsed = defaultListLimit
		}
		limit = parsed
	}

	results, err := h.svc.SearchWork(r.Context(), query, limit)
	if err != nil {
		if det.Representation == represent.RepresentationHTML {
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "search failed"})
		return
	}

	rows := make([]WorkListRow, 0, len(results))
	for _, result := range results {
		rows = append(rows, WorkListRow{
			ID:         result.ID,
			Title:      result.Title,
			Kind:       result.Kind,
			Visibility: result.Visibility,
			Status:     result.Status,
			CreatedAt:  result.CreatedAt,
		})
	}

	if det.Representation != represent.RepresentationHTML {
		writeJSON(w, http.StatusOK, map[string]any{
			"query":  query,
			"limit":  limit,
			"items":  rows,
			"count":  len(rows),
		})
		return
	}

	// For HTML, render a search page
	data := searchPageData{
		NavActive: "search",
		Query:     query,
		Results:   rows,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := searchTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

type searchPageData struct {
	NavActive string
	Query     string
	Results   []WorkListRow
}
