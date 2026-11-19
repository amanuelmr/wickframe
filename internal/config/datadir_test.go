// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultDataDirFollowsThePlatform(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("LocalAppData", filepath.Join(root, "local"))

	want := filepath.Join(root, "data", "wickframe")
	switch runtime.GOOS {
	case "windows":
		want = filepath.Join(root, "local", "Wickframe")
	case "darwin":
		want = filepath.Join(root, "Library", "Application Support", "Wickframe")
	}
	got, err := DefaultDataDir()
	if err != nil || got != want {
		t.Errorf("DefaultDataDir() = %q, %v, want %q", got, err, want)
	}
}
