// SPDX-License-Identifier: AGPL-3.0-only

package secret_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/amanuelmr/wickframe/internal/platform/secret"
)

var binding = secret.Binding{Table: "data_sources", Column: "credentials", RowID: "0192f0c8-7b2a-7000-8000-000000000001"}

func keyring(t *testing.T, current []byte, previous ...[]byte) *secret.Keyring {
	t.Helper()
	k, err := secret.NewKeyring(current, previous...)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestSealThenOpenRoundTrips(t *testing.T) {
	t.Parallel()
	key := secret.GenerateKey()
	k := keyring(t, key)
	sealed := k.Seal([]byte("api-key-123"), binding)
	if want := "v1:" + secret.KeyID(key) + ":"; !strings.HasPrefix(sealed, want) {
		t.Fatalf("sealed = %q, want prefix %q", sealed, want)
	}
	if strings.Contains(sealed, "api-key-123") {
		t.Fatalf("sealed value contains the plaintext: %q", sealed)
	}
	got, err := k.Open(sealed, binding)
	if err != nil || string(got) != "api-key-123" {
		t.Fatalf("Open() = %q, %v; want the plaintext", got, err)
	}
	if again := k.Seal([]byte("api-key-123"), binding); again == sealed {
		t.Fatal("sealing the same value twice gave the same output; nonces are not random")
	}
}

func TestOpenRejectsAnotherRowsBinding(t *testing.T) {
	t.Parallel()
	k := keyring(t, secret.GenerateKey())
	sealed := k.Seal([]byte("api-key-123"), binding)
	other := binding
	other.RowID = "0192f0c8-7b2a-7000-8000-000000000002"
	if _, err := k.Open(sealed, other); !errors.Is(err, secret.ErrInvalid) {
		t.Fatalf("Open() with another row's binding = %v, want ErrInvalid", err)
	}
}

func TestBindingsThatJoinToTheSameStringDoNotOpenEachOther(t *testing.T) {
	t.Parallel()
	k := keyring(t, secret.GenerateKey())
	a := secret.Binding{Table: "a:b", Column: "c", RowID: "d"}
	b := secret.Binding{Table: "a", Column: "b:c", RowID: "d"}
	if _, err := k.Open(k.Seal([]byte("x"), a), b); !errors.Is(err, secret.ErrInvalid) {
		t.Fatalf("a value sealed for %+v opened for %+v: %v", a, b, err)
	}
	if _, err := k.Open(k.Seal([]byte("x"), b), a); !errors.Is(err, secret.ErrInvalid) {
		t.Fatalf("a value sealed for %+v opened for %+v: %v", b, a, err)
	}
}

func TestOpenRejectsTamperedOrMalformedValues(t *testing.T) {
	t.Parallel()
	k := keyring(t, secret.GenerateKey())
	sealed := k.Seal([]byte("api-key-123"), binding)
	parts := strings.Split(sealed, ":")
	flipped := []byte(parts[3])
	flipped[0] ^= 1
	for name, value := range map[string]string{
		"flipped ciphertext": strings.Join([]string{parts[0], parts[1], parts[2], string(flipped)}, ":"),
		"truncated":          sealed[:len(sealed)-4],
		"other version":      "v2" + sealed[2:],
		"missing part":       strings.Join(parts[:3], ":"),
		"short nonce":        strings.Join([]string{parts[0], parts[1], "AAAA", parts[3]}, ":"),
		"bad base64":         strings.Join([]string{parts[0], parts[1], parts[2], "!!"}, ":"),
		"empty":              "",
	} {
		if _, err := k.Open(value, binding); !errors.Is(err, secret.ErrInvalid) {
			t.Errorf("%s: Open() = %v, want ErrInvalid", name, err)
		}
	}
}

func TestRotationKeepsOldValuesReadable(t *testing.T) {
	t.Parallel()
	oldKey, newKey := secret.GenerateKey(), secret.GenerateKey()
	sealedOld := keyring(t, oldKey).Seal([]byte("old"), binding)

	rotating := keyring(t, newKey, oldKey)
	if got, err := rotating.Open(sealedOld, binding); err != nil || string(got) != "old" {
		t.Fatalf("Open() of a value sealed with the previous key = %q, %v", got, err)
	}
	if sealed := rotating.Seal([]byte("new"), binding); !strings.HasPrefix(sealed, "v1:"+secret.KeyID(newKey)+":") {
		t.Fatalf("Seal() during rotation used %q, want the new key", sealed)
	}
	if _, err := keyring(t, newKey).Open(sealedOld, binding); !errors.Is(err, secret.ErrUnknownKey) {
		t.Fatalf("Open() after the old key is dropped = %v, want ErrUnknownKey", err)
	}
}

func TestNewKeyringRejectsWrongKeySizes(t *testing.T) {
	t.Parallel()
	for _, key := range [][]byte{nil, make([]byte, 16), make([]byte, secret.KeySize+1)} {
		if _, err := secret.NewKeyring(key); err == nil {
			t.Errorf("NewKeyring with a %d-byte key succeeded, want error", len(key))
		}
		if _, err := secret.NewKeyring(secret.GenerateKey(), key); err == nil {
			t.Errorf("NewKeyring with a %d-byte previous key succeeded, want error", len(key))
		}
	}
}

func FuzzOpen(f *testing.F) {
	k, err := secret.NewKeyring(make([]byte, secret.KeySize))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(k.Seal([]byte("x"), binding))
	f.Add("v1:00000000::")
	f.Fuzz(func(t *testing.T, s string) {
		if _, err := k.Open(s, binding); err == nil && !strings.HasPrefix(s, "v1:") {
			t.Fatalf("Open(%q) succeeded without the v1 prefix", s)
		}
	})
}
