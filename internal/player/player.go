package player

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Player struct {
	ID          string    `json:"id" bson:"_id"`
	DisplayName string    `json:"display_name,omitempty" bson:"display_name,omitempty"`
	Country     string    `json:"country,omitempty" bson:"country,omitempty"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// Patch is a partial update.
// Use pointers to know when field is empty or not sent.
type Patch struct {
	DisplayName *string `json:"display_name"`
	Country     *string `json:"country"`
}

var (
	ErrInvalidID    = errors.New("invalid player id")
	ErrNotFound     = errors.New("player not found")
	ErrInvalidPatch = errors.New("invalid patch")
)

var (
	countryPattern = regexp.MustCompile(`^[A-Z]{2}$`) // ISO 3166-1 alpha-2
	idPattern      = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
)

func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return ErrInvalidID
	}
	return nil
}

func (p Patch) validate() error {
	if p.DisplayName == nil && p.Country == nil {
		return fmt.Errorf("%w: no fields to update", ErrInvalidPatch)
	}
	if p.DisplayName != nil {
		n := utf8.RuneCountInString(strings.TrimSpace(*p.DisplayName))
		if n < 1 || n > 32 {
			return fmt.Errorf("%w: display_name must be 1-32 characters", ErrInvalidPatch)
		}
	}
	if p.Country != nil && !countryPattern.MatchString(*p.Country) {
		return fmt.Errorf("%w: country must be ISO 3166-1 alpha-2", ErrInvalidPatch)
	}
	return nil
}
