package packets

import (
	"strings"
)

type ValidationErrors map[string]string

func (v ValidationErrors) Add(field, message string) {
	if _, exists := v[field]; exists {
		return
	}
	v[field] = message
}

func (v ValidationErrors) Empty() bool {
	return len(v) == 0
}

func NormalizeDraft(in DraftInput) NormalizedPacket {
	kind := normalizeKind(in.Kind)
	visibility := normalizeVisibility(in.Visibility)

	// Safety default: private security items must never become public by intake default.
	if kind == KindPrivateSecurity && visibility != VisibilityPrivate {
		visibility = VisibilityPrivate
	}

	return NormalizedPacket{
		Title:              strings.TrimSpace(in.Title),
		Kind:               kind,
		Scope:              normalizeList(in.Scope),
		Deliverables:       normalizeList(in.Deliverables),
		AcceptanceCriteria: normalizeList(in.AcceptanceCriteria),
		RewardModel:        strings.TrimSpace(in.RewardModel),
		Visibility:         visibility,
	}
}

func ValidateDraftInput(in DraftInput) ValidationErrors {
	errs := ValidationErrors{}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		errs.Add("title", "Title is required.")
	}

	if !isKind(strings.TrimSpace(strings.ToLower(in.Kind))) {
		errs.Add("kind", "Kind must be one of: bounty, rfq, rfp, private_security.")
	}

	if !isVisibility(strings.TrimSpace(strings.ToLower(in.Visibility))) {
		errs.Add("visibility", "Visibility must be one of: draft, private, public, archived.")
	}

	return errs
}

func normalizeKind(raw string) Kind {
	switch Kind(strings.TrimSpace(strings.ToLower(raw))) {
	case KindBounty:
		return KindBounty
	case KindRFQ:
		return KindRFQ
	case KindRFP:
		return KindRFP
	case KindPrivateSecurity:
		return KindPrivateSecurity
	default:
		return KindBounty
	}
}

func isKind(raw string) bool {
	switch Kind(raw) {
	case KindBounty, KindRFQ, KindRFP, KindPrivateSecurity:
		return true
	default:
		return false
	}
}

func normalizeVisibility(raw string) Visibility {
	switch Visibility(strings.TrimSpace(strings.ToLower(raw))) {
	case VisibilityDraft:
		return VisibilityDraft
	case VisibilityPrivate:
		return VisibilityPrivate
	case VisibilityPublic:
		return VisibilityPublic
	case VisibilityArchived:
		return VisibilityArchived
	default:
		return VisibilityDraft
	}
}

func isVisibility(raw string) bool {
	switch Visibility(raw) {
	case VisibilityDraft, VisibilityPrivate, VisibilityPublic, VisibilityArchived:
		return true
	default:
		return false
	}
}

func normalizeList(raw string) []string {
	normalizedRaw := strings.ReplaceAll(raw, "\r\n", "\n")
	normalizedRaw = strings.ReplaceAll(normalizedRaw, `\n`, "\n")
	rows := strings.Split(normalizedRaw, "\n")
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		v := strings.TrimSpace(row)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}
