package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/config"
)

func newEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "Print resolved configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			timeout := cfg.Timeout.String()
			insecure := "false"
			if cfg.Insecure {
				insecure = "true"
			}
			caCert := cfg.CACert
			if caCert == "" {
				caCert = "(none)"
			}
			configFile := config.ConfigFilePath()
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				configFile += " (not found)"
			}

			fmt.Printf("server:      %s\n", cfg.ServerURL)
			fmt.Printf("source:      %s\n", cfg.Source)
			fmt.Printf("timeout:     %s\n", timeout)
			fmt.Printf("insecure:    %s\n", insecure)
			fmt.Printf("ca_cert:     %s\n", caCert)
			fmt.Printf("config_file: %s\n", configFile)
			return nil
		},
	}
}
