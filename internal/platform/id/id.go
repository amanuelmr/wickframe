// SPDX-License-Identifier: AGPL-3.0-only

// Package id generates and parses UUIDv7 identifiers (RFC 9562). The 16 bytes
// are in network order with the millisecond timestamp first, which keeps them
// index-friendly; both SQL dialects store exactly these bytes (ADR-0004).
package id

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"
)

// ID is a UUIDv7.
type ID [16]byte

// Nil is the zero ID.
var Nil ID

// ErrInvalid is returned by Parse for malformed input.
var ErrInvalid = errors.New("invalid uuid")

// New returns a UUIDv7 for time now. Pass the injected clock's time, never
// time.Now(), so tests stay deterministic in their ordering.
func New(now time.Time) ID {
	var u ID
	_, _ = rand.Read(u[6:]) // never fails since Go 1.24

	var ms [8]byte
	binary.BigEndian.PutUint64(ms[:], uint64(now.UnixMilli()))
	copy(u[:6], ms[2:])     // the low 48 bits, big-endian
	u[6] = u[6]&0x0f | 0x70 // version 7
	u[8] = u[8]&0x3f | 0x80 // RFC 9562 variant
	return u
}

// Parse parses the canonical 36-character form.
func Parse(s string) (ID, error) {
	var u ID
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return Nil, ErrInvalid
	}
	hexs := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	if _, err := hex.Decode(u[:], []byte(hexs)); err != nil {
		return Nil, ErrInvalid
	}
	return u, nil
}

// String returns the canonical lower-case form.
func (u ID) String() string {
	var b [36]byte
	hex.Encode(b[0:8], u[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], u[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], u[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], u[8:10])
	b[23] = '-'
	hex.Encode(b[24:], u[10:])
	return string(b[:])
}

// Time returns the millisecond timestamp embedded in a UUIDv7.
func (u ID) Time() time.Time {
	ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 | int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])
	return time.UnixMilli(ms).UTC()
}
