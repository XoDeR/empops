// Package uuidv7 provides UUID v7 (time-ordered) identifier generation,
// used as the preferred primary key format across EmpOps Go tables.
package uuidv7

import "github.com/google/uuid"

// New returns a new UUID v7 value as a string.
func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		// uuid.NewV7 only fails if the system clock/entropy source is
		// broken; fall back to a random v4 rather than panicking.
		return uuid.NewString()
	}
	return id.String()
}

// NewUUID returns a new UUID v7 value as a uuid.UUID.
func NewUUID() (uuid.UUID, error) {
	return uuid.NewV7()
}
