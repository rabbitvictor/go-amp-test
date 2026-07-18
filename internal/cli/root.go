package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// Version is the CLI version, overridable at build time via -ldflags.
var Version = "0.1.0"

// usageError marks an error caused by bad CLI usage (wrong args, invalid
// flags). Execute maps these to exit code 2; all other errors exit 1.
type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

// newUsageError wraps err as a usage error.
func newUsageError(err error) error { return &usageError{err: err} }

// rootFlags holds the persistent flags shared by all subcommands.
type rootFlags struct {
	server  string
	timeout time.Duration
	format  string
}

// newRoot builds the root command and its persistent flags.
func newRoot(out, errOut io.Writer) (*cobra.Command, *rootFlags) {
	flags := &rootFlags{}

	root := &cobra.Command{
		Use:           "go-amp-test",
		Short:         "Client for the go-amp-test web service",
		Long:          "go-amp-test is a CLI client for the go-amp-test web service.\nIt talks to the HTTP API and prints JSON results to stdout.",
		SilenceUsage:  true, // usage is printed by Execute on usage errors only
		SilenceErrors: true, // we print errors ourselves with context
	}
	root.SetOut(out)
	root.SetErr(errOut)

	root.PersistentFlags().StringVarP(
		&flags.server, "server", "s", defaultServerURL(),
		"base URL of the web service (env: GO_AMP_SERVER)",
	)
	root.PersistentFlags().DurationVarP(
		&flags.timeout, "timeout", "t", 30*time.Second,
		"HTTP request timeout",
	)
	root.PersistentFlags().StringVar(
		&flags.format, "format", string(formatJSON),
		"output format: json|compact",
	)

	// Pre-run validation shared by commands that hit the server.
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		switch outputFormat(flags.format) {
		case formatJSON, formatCompact:
		default:
			return newUsageError(fmt.Errorf("invalid --format %q (want json|compact)", flags.format))
		}
		return nil
	}

	root.AddCommand(newHealthCmd(flags, out, errOut))
	root.AddCommand(newItemsCmd(flags, out, errOut))
	root.AddCommand(newVersionCmd(out))
	return root, flags
}

// newAPIClient builds a Client from the resolved root flags.
func (f *rootFlags) newAPIClient() *Client {
	return NewClient(f.server, f.timeout)
}

// defaultServerURL returns the base URL from GO_AMP_SERVER if set, else the
// local default.
func defaultServerURL() string {
	if v := os.Getenv("GO_AMP_SERVER"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func (f *rootFlags) outFormat() outputFormat { return outputFormat(f.format) }

// Execute runs the root command and returns the process exit code. It prints
// usage errors (exit 2) and runtime errors (exit 1) to stderr.
func Execute() int {
	root, _ := newRoot(os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		var ue *usageError
		if errors.As(err, &ue) {
			return 2
		}
		return 1
	}
	return 0
}
