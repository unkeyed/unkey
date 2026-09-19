package cdc

import (
	"encoding/json"
	"errors"
)

// resumeToken stores a stream position with its filter rules.
// This prevents resuming with different rules. It does not grant access.
type resumeToken struct {
	Rules    []Rule          `json:"rules"`
	Position json.RawMessage `json:"position"`
}

// ErrInvalidToken means the token cannot be read or does not match this watch.
// Discard it and start a new snapshot.
var ErrInvalidToken = errors.New("invalid CDC resume token")

// ErrExpired means the saved binlog position is no longer available.
// Start a new snapshot and remove destination records that no longer exist.
var ErrExpired = errors.New("CDC resume position expired")
