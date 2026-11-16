// SPDX-License-Identifier: AGPL-3.0-only

// Command wickframe is the Wickframe server and CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "wickframe",
		Short:        "Self-hosted charting and alerting for market data",
		SilenceUsage: true,
	}
	root.AddCommand(
		&cobra.Command{
			Use:   "version",
			Short: "Print the version",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "wickframe", version)
				return err
			},
		},
		&cobra.Command{
			Use:   "serve",
			Short: "Run the Wickframe server",
			Args:  cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				return errors.New("serve is not implemented yet")
			},
		},
	)
	return root
}
