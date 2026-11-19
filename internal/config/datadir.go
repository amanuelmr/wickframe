// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultDataDir returns the per-user directory for Wickframe's database:
// Application Support on macOS, %LocalAppData% on Windows and
// $XDG_DATA_HOME (or ~/.local/share) elsewhere.
func DefaultDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// Not %AppData%, which roaming profiles copy between machines.
		if dir := os.Getenv("LocalAppData"); dir != "" {
			return filepath.Join(dir, "Wickframe"), nil
		}
		return "", errors.New("data directory: %LocalAppData% is not set")
	case "darwin":
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("data directory: %w", err)
		}
		return filepath.Join(dir, "Wickframe"), nil
	default:
		if dir := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dir) {
			return filepath.Join(dir, "wickframe"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("data directory: %w", err)
		}
		return filepath.Join(home, ".local", "share", "wickframe"), nil
	}
}
