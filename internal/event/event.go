package event

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Event uses a mongo bson type in its ID field for simplicty
type Event struct {
	ID         bson.ObjectID  `json:"id"                bson:"_id,omitempty"`
	PlayerID   string         `json:"player_id"         bson:"player_id"`
	Type       string         `json:"type"              bson:"type"`
	Timestamp  time.Time      `json:"timestamp"         bson:"ts"`          // when it happened (client clock)
	RecordedAt time.Time      `json:"recorded_at"       bson:"recorded_at"` // when we received it (server clock)
	Payload    map[string]any `json:"payload,omitempty" bson:"payload,omitempty"`
}

// RecordRequest is the wire DTO. Timestamp is optional: absent means "now".
type RecordRequest struct {
	Type      string         `json:"type"`
	Timestamp *time.Time     `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

// Filter bounds a recent-events read.
type Filter struct {
	Type  string // optional exact match
	Limit int    // 0 → DefaultLimit, capped at MaxLimit
}

const (
	DefaultLimit = 20
	MaxLimit     = 100
	maxClockSkew = 5 * time.Minute
)

var ErrInvalidEvent = errors.New("invalid event")

var typePattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,64}$`)

func (r RecordRequest) validate(now time.Time) error {
	if !typePattern.MatchString(r.Type) {
		return fmt.Errorf("%w: type must be 1-64 chars of [a-zA-Z0-9_.:-]", ErrInvalidEvent)
	}
	if r.Timestamp != nil && r.Timestamp.After(now.Add(maxClockSkew)) {
		return fmt.Errorf("%w: timestamp is in the future", ErrInvalidEvent)
	}
	return nil
}

func (f Filter) normalized() Filter {
	switch {
	case f.Limit <= 0:
		f.Limit = DefaultLimit
	case f.Limit > MaxLimit:
		f.Limit = MaxLimit
	}
	return f
}
