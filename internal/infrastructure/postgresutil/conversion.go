// Package postgresutil provides shared SQL NULL <-> Go sentinel conversion
// helpers for the postgres repository implementations.
package postgresutil

import (
	"time"

	"github.com/google/uuid"
)

// NullIfZeroUUID converts the nil-UUID empty sentinel into a SQL NULL.
func NullIfZeroUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// NullIfZeroTime converts the zero-time empty sentinel into a SQL NULL.
func NullIfZeroTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// NullIfZeroInt converts the zero int empty sentinel into a SQL NULL.
func NullIfZeroInt(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

// ValueOrNilUUID converts a scanned SQL NULL into the nil-UUID sentinel.
func ValueOrNilUUID(p *uuid.UUID) uuid.UUID {
	if p == nil {
		return uuid.Nil
	}
	return *p
}

// ValueOrNilTime converts a scanned SQL NULL into the zero-time sentinel.
func ValueOrNilTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}

// ValueOrNilInt converts a scanned SQL NULL into the zero int sentinel.
func ValueOrNilInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
