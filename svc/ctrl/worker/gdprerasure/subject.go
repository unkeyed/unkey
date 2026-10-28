package gdprerasure

import (
	"fmt"
	"regexp"
)

// subject holds the values a subject's steps bind into their clauses.
type subject struct {
	workspaceID      string
	externalID       string
	userID           string
	email            string
	identityIDs      []string
	orgID            string
	stripeCustomerID string
}

var placeholderPattern = regexp.MustCompile(`:([a-z_]+)`)

// bind replaces each :name placeholder in clause with a positional ? and
// returns the matching values. Registry clauses are compile-time constants, so
// an unknown or empty placeholder is a programmer error, never input.
func (s subject) bind(clause string) (string, []any, error) {
	var args []any
	var bindErr error
	query := placeholderPattern.ReplaceAllStringFunc(clause, func(match string) string {
		value, err := s.value(match[1:])
		if err != nil && bindErr == nil {
			bindErr = err
		}
		args = append(args, value)
		return "?"
	})
	if bindErr != nil {
		return "", nil, bindErr
	}
	return query, args, nil
}

func (s subject) value(name string) (any, error) {
	scalars := map[string]string{
		"workspace_id": s.workspaceID,
		"external_id":  s.externalID,
		"user_id":      s.userID,
	}
	if name == "identity_ids" {
		return s.identityIDs, nil
	}
	value, known := scalars[name]
	if !known {
		return nil, fmt.Errorf("unknown placeholder :%s", name)
	}
	if value == "" {
		return nil, fmt.Errorf("placeholder :%s is empty for this subject", name)
	}
	return value, nil
}
