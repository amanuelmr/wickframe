// SPDX-License-Identifier: AGPL-3.0-only

// Command wickframe is the Wickframe server and CLI.
package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/amanuelmr/wickframe/internal/app"
	"github.com/amanuelmr/wickframe/internal/config"
	"github.com/amanuelmr/wickframe/internal/platform/logx"
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
		newServeCmd(),
		&cobra.Command{
			Use:   "openapi",
			Short: "Print the HTTP API's OpenAPI document as YAML",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				doc, err := app.OpenAPI()
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(doc)
				return err
			},
		},
	)
	return root
}

func newServeCmd() *cobra.Command {
	var addr, dataDir string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the Wickframe server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			log, err := logx.New(cmd.ErrOrStderr(), logx.FormatAuto, slog.LevelInfo)
			if err != nil {
				return err //nolint:wrapcheck // already says what failed
			}
			if dataDir == "" {
				if dataDir, err = config.DefaultDataDir(); err != nil {
					return err
				}
			}
			ln, err := new(net.ListenConfig).Listen(ctx, "tcp", addr)
			if err != nil {
				return fmt.Errorf("listen: %w", err)
			}
			log.Info("serving", "url", "http://"+ln.Addr().String(), "data_dir", dataDir)
			if err := app.Run(ctx, ln, dataDir, log); err != nil {
				return err //nolint:wrapcheck // app errors carry their own context
			}
			log.Info("stopped")
			return nil
		},
	}
	cmd.Flags().StringVar(&addr, "addr", app.DefaultAddr, "address to listen on")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "directory for the database (default: the user's data directory)")
	return cmd
}
