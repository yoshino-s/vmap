package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"vmap/internal/logx"
	"vmap/internal/scanner"
)

const (
	modeAuto  = "auto"
	modeHost  = "host"
	modeGuest = "guest"
)

var opts scanner.Options

var rootCmd = &cobra.Command{
	Use:   "vmap",
	Short: "Scan vsock CIDs and ports",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateMode(opts.Mode); err != nil {
			return err
		}
		if err := validateLogLevel(opts.LogLevel); err != nil {
			return err
		}

		s, err := scanner.New(opts)
		if err != nil {
			return err
		}

		return s.Run(cmd.Context())
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&opts.Mode, "mode", modeAuto, "scan mode: host, guest, auto")
	rootCmd.Flags().StringVar(&opts.CIDInput, "cid", "", "CID list/range/all, examples: 2,3,5-10,all")
	rootCmd.Flags().StringVar(&opts.PortInput, "port", "all", "port list/range/all, examples: 22,80,1000-1010,all")
	rootCmd.Flags().StringSliceVar(&opts.Payloads, "payload", nil, "custom detect payload(s), repeat or comma-separated; supports escapes like \\n and token empty")
	rootCmd.Flags().BoolVar(&opts.Detect, "detect", false, "send common payloads and print non-empty responses")
	rootCmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "per-connection timeout, 0 means no timeout")
	rootCmd.Flags().DurationVar(&opts.Interval, "interval", 0, "sleep interval between probes, 0 means no interval")
	rootCmd.Flags().StringVar(&opts.LogLevel, "log-level", "info", "log level: debug, info, warn, error")
}

func validateMode(mode string) error {
	switch mode {
	case modeAuto, modeHost, modeGuest:
		return nil
	default:
		return fmt.Errorf("invalid --mode %q, expected one of: %s|%s|%s", mode, modeHost, modeGuest, modeAuto)
	}
}

func validateLogLevel(level string) error {
	_, err := logx.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid --log-level: %w", err)
	}
	return nil
}
