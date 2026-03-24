package packets

import "time"

// Kind is the normalized work item kind.
type Kind string

const (
	KindBounty          Kind = "bounty"
	KindRFQ             Kind = "rfq"
	KindRFP             Kind = "rfp"
	KindPrivateSecurity Kind = "private_security"
)

// Visibility is the normalized visibility intent for a packet.
type Visibility string

const (
	VisibilityDraft    Visibility = "draft"
	VisibilityPrivate  Visibility = "private"
	VisibilityPublic   Visibility = "public"
	VisibilityArchived Visibility = "archived"
)

// DraftInput represents raw intake from the draft form.
type DraftInput struct {
	Title              string
	Kind               string
	Scope              string
	Deliverables       string
	AcceptanceCriteria string
	RewardModel        string
	Visibility         string
}

// NormalizedPacket is the deterministic packet shape used for preview/versioning.
type NormalizedPacket struct {
	Title              string
	Kind               Kind
	Scope              []string
	Deliverables       []string
	AcceptanceCriteria []string
	RewardModel        string
	Visibility         Visibility
	NormalizedAt       time.Time
}
