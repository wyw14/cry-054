package domain

import (
	"fmt"
	"strings"
	"time"
)

type ActorRole string

const (
	RoleClaimant ActorRole = "claimant"
	RoleReviewer ActorRole = "reviewer"
	RoleManager  ActorRole = "manager"
	RoleAuditor  ActorRole = "auditor"
)

type ReviewAction string

const (
	ReviewAccept     ReviewAction = "accept"
	ReviewReject     ReviewAction = "reject"
	ReviewSupplement ReviewAction = "request_supplement"
	ReviewException  ReviewAction = "approve_exception"
	ReviewReopen     ReviewAction = "reopen"
)

type ReviewDecision struct {
	ID        string            `json:"id"`
	ClaimID   string            `json:"claim_id"`
	ActorID   string            `json:"actor_id"`
	ActorRole ActorRole         `json:"actor_role"`
	Action    ReviewAction      `json:"action"`
	Reason    string            `json:"reason"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

func ApplyReview(claim *ExpenseClaim, decision ReviewDecision, expectedVersion int64, now time.Time) error {
	if claim == nil {
		return fmt.Errorf("claim is nil: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(decision.ActorID) == "" || strings.TrimSpace(decision.Reason) == "" {
		return fmt.Errorf("review actor and reason are required: %w", ErrInvalidInput)
	}
	if decision.ActorRole != RoleReviewer && decision.ActorRole != RoleManager {
		return fmt.Errorf("role %s cannot review: %w", decision.ActorRole, ErrForbidden)
	}
	var next ClaimStatus
	switch decision.Action {
	case ReviewAccept:
		next = ClaimApproved
	case ReviewReject:
		next = ClaimRejected
	case ReviewSupplement:
		next = ClaimNeedSupplement
	case ReviewException:
		if decision.ActorRole != RoleManager {
			return fmt.Errorf("exception approval requires manager: %w", ErrForbidden)
		}
		next = ClaimApproved
	case ReviewReopen:
		if decision.ActorRole != RoleManager {
			return fmt.Errorf("reopen requires manager: %w", ErrForbidden)
		}
		next = ClaimUnderReview
	default:
		return fmt.Errorf("unknown review action %s: %w", decision.Action, ErrInvalidInput)
	}
	return claim.Transition(next, expectedVersion, now)
}

type ReviewQueueItem struct {
	Claim           ExpenseClaim `json:"claim"`
	RiskFlags       []string     `json:"risk_flags"`
	DaysWaiting     int          `json:"days_waiting"`
	SuggestedAction string       `json:"suggested_action"`
}
