// SPDX-License-Identifier: AGPL-3.0-only

// Package secret encrypts secrets at rest with XChaCha20-Poly1305
// (11-security §5). Sealed values look like v1:<keyID>:<nonce>:<ciphertext>,
// and each one is bound to the row it belongs to, so ciphertexts cannot be
// swapped between rows.
package secret

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
)

// KeySize is the master key size in bytes (256 bits).
const KeySize = chacha20poly1305.KeySize

var (
	// ErrUnknownKey means the value was sealed with a key the Keyring does not hold.
	ErrUnknownKey = errors.New("secret sealed with an unknown key")
	// ErrInvalid means the value is malformed, tampered with, or bound to another row.
	ErrInvalid = errors.New("invalid sealed secret")
)

var b64 = base64.RawURLEncoding

// Keyring seals with its current key and opens with any of its keys, so values
// stay readable while a key rotation is in progress.
type Keyring struct {
	currentID string
	aeads     map[string]cipher.AEAD
}

// GenerateKey returns a new random master key.
func GenerateKey() []byte {
	key := make([]byte, KeySize)
	_, _ = rand.Read(key) // never fails since Go 1.24
	return key
}

// KeyID identifies a key in sealed values: the first 8 hex characters of its
// SHA-256, so keys need no separate naming.
func KeyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:4])
}

// NewKeyring returns a Keyring that seals with current and can also open values
// sealed with any of previous.
func NewKeyring(current []byte, previous ...[]byte) (*Keyring, error) {
	if len(current) == 0 {
		return nil, errors.New("no current secret key")
	}
	k := &Keyring{currentID: KeyID(current), aeads: make(map[string]cipher.AEAD)}
	keys := make(map[string][]byte)
	for _, key := range append([][]byte{current}, previous...) {
		if len(key) != KeySize {
			return nil, fmt.Errorf("secret key is %d bytes, want %d", len(key), KeySize)
		}
		id := KeyID(key)
		if seen, ok := keys[id]; ok && !bytes.Equal(seen, key) {
			return nil, fmt.Errorf("two different secret keys share key ID %s", id)
		}
		keys[id] = key
		aead, err := chacha20poly1305.NewX(key)
		if err != nil {
			return nil, fmt.Errorf("secret key: %w", err)
		}
		k.aeads[id] = aead
	}
	return k, nil
}

// Binding names where a sealed value is stored. It is the associated data of
// the ciphertext, so a value only opens for the row it was sealed for.
type Binding struct {
	Table, Column, RowID string
}

// Fields are length-prefixed, so values cannot shift from one field to another
// (Table "a:b", Column "c" differs from Table "a", Column "b:c").
func (b Binding) bytes() []byte {
	var out []byte
	for _, f := range []string{b.Table, b.Column, b.RowID} {
		out = binary.AppendUvarint(out, uint64(len(f)))
		out = append(out, f...)
	}
	return out
}

// Seal encrypts plaintext with the current key, bound to b.
func (k *Keyring) Seal(plaintext []byte, b Binding) string {
	aead := k.aeads[k.currentID]
	nonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce) // 192-bit random nonces are safe at any volume
	ciphertext := aead.Seal(nil, nonce, plaintext, b.bytes())
	return "v1:" + k.currentID + ":" + b64.EncodeToString(nonce) + ":" + b64.EncodeToString(ciphertext)
}

// Open decrypts a value from Seal. It returns ErrUnknownKey when the key is not
// in the Keyring and ErrInvalid for anything else that does not authenticate.
func (k *Keyring) Open(sealed string, b Binding) ([]byte, error) {
	parts := strings.Split(sealed, ":")
	if len(parts) != 4 || parts[0] != "v1" {
		return nil, ErrInvalid
	}
	aead, ok := k.aeads[parts[1]]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownKey, parts[1])
	}
	nonce, err := b64.DecodeString(parts[2])
	if err != nil || len(nonce) != aead.NonceSize() {
		return nil, ErrInvalid
	}
	ciphertext, err := b64.DecodeString(parts[3])
	if err != nil {
		return nil, ErrInvalid
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, b.bytes())
	if err != nil {
		return nil, ErrInvalid
	}
	return plaintext, nil
}
