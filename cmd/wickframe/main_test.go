// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "prints version", args: []string{"version"}, want: "wickframe dev\n"},
		{name: "rejects arguments", args: []string{"version", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := newRootCmd()
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tt.args)

			err := cmd.ExecuteContext(t.Context())
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := out.String(); !tt.wantErr && got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}
