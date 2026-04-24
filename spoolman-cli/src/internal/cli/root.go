package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/api"
	"github.com/vibecoder/spoolctl/internal/config"
)

var (
	flagServer  string
	flagVerbose bool
	flagQuiet   bool
	flagTimeout string
)

// rootCmd is the top-level cobra command.
var rootCmd = &cobra.Command{
	Use:   "spoolctl",
	Short: "CLI for the Spoolman filament inventory API",
	Long: `spoolctl is a CLI for Spoolman (self-hosted 3D printer filament tracker).

Server resolution order:
  1. --server flag
  2. SPOOLMAN_URL env var
  3. ~/.config/spoolctl/config.toml (server = "...")
  4. http://localhost:7912/api/v1 (default)

Extra env vars:
  SPOOLMAN_TIMEOUT   request timeout (default: 10s)
  SPOOLMAN_INSECURE  skip TLS verify (set to 1)
  SPOOLMAN_CA_CERT   path to custom CA bundle`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		writeError(err, 1)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagServer, "server", "", "Spoolman server URL (overrides env and config)")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output (shows provenance)")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress informational output")
	rootCmd.PersistentFlags().StringVar(&flagTimeout, "timeout", "", "Request timeout (e.g. 10s)")

	rootCmd.AddCommand(newEnvCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newHealthCmd())
	rootCmd.AddCommand(newContextCmd())
	rootCmd.AddCommand(newVendorCmd())
	rootCmd.AddCommand(newFilamentCmd())
	rootCmd.AddCommand(newSpoolCmd())
	rootCmd.AddCommand(newDBCmd())
	rootCmd.AddCommand(newCompletionCmd())
}

// loadConfig applies --timeout flag override then calls config.Load.
func loadConfig() (*config.Config, error) {
	if flagTimeout != "" {
		_ = os.Setenv("SPOOLMAN_TIMEOUT", flagTimeout)
	}
	return config.Load(flagServer)
}

// newClient creates an API client from current flags.
func newClient() (*api.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return api.NewClient(cfg, flagVerbose)
}

// printJSON prints v as indented JSON to stdout.
func printJSON(v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// printRaw prints raw JSON to stdout.
func printRaw(b []byte) {
	// Pretty-print if possible
	var v interface{}
	if json.Unmarshal(b, &v) == nil {
		if out, err := json.MarshalIndent(v, "", "  "); err == nil {
			fmt.Println(string(out))
			return
		}
	}
	fmt.Println(string(b))
}

// writeError writes a JSON error to stderr and exits.
func writeError(err error, status int) {
	type errOut struct {
		Error  string `json:"error"`
		Status int    `json:"status"`
	}
	b, _ := json.Marshal(errOut{Error: err.Error(), Status: status})
	fmt.Fprintln(os.Stderr, string(b))
}

// die writes an error to stderr and exits 1.
func die(err error) {
	writeError(err, 1)
	os.Exit(1)
}

// ptr helpers
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func f64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}
func boolPtr(b bool) *bool { return &b }
