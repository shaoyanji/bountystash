package handlers

import (
	"bytes"
	"testing"

	"github.com/shaoyanji/bountystash/internal/packets"
)

func TestCanonicalJSONDeterministic(t *testing.T) {
	packet := packets.NormalizedPacket{
		Title:              "Deterministic Hashing",
		Kind:               packets.KindBounty,
		Scope:              []string{"a", "b"},
		Deliverables:       []string{"patch"},
		AcceptanceCriteria: []string{"tests pass"},
		RewardModel:        "fixed",
		Visibility:         packets.VisibilityDraft,
	}

	first, err := canonicalJSON(packet)
	if err != nil {
		t.Fatalf("canonicalJSON first: %v", err)
	}
	second, err := canonicalJSON(packet)
	if err != nil {
		t.Fatalf("canonicalJSON second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("canonical JSON changed between calls:\n%s\n%s", first, second)
	}
}

func TestQuotientProjectionExcludesTitle(t *testing.T) {
	left := packets.NormalizedPacket{
		Title:              "One",
		Kind:               packets.KindRFQ,
		Scope:              []string{"scope"},
		Deliverables:       []string{"deliver"},
		AcceptanceCriteria: []string{"accept"},
		RewardModel:        "quote",
		Visibility:         packets.VisibilityPublic,
	}
	right := left
	right.Title = "Two"

	leftProjection, err := quotientProjectionJSON(left)
	if err != nil {
		t.Fatalf("left projection: %v", err)
	}
	rightProjection, err := quotientProjectionJSON(right)
	if err != nil {
		t.Fatalf("right projection: %v", err)
	}

	if !bytes.Equal(leftProjection, rightProjection) {
		t.Fatalf("expected same projection when title differs:\n%s\n%s", leftProjection, rightProjection)
	}
}
