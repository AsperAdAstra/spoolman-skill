package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/config"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Print Spoolman server info",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			info, err := c.GetInfo()
			if err != nil {
				return err
			}
			if info.Version != config.TestedVersion && !flagQuiet {
				fmt.Printf("# warning: tested against %s, server is %s\n", config.TestedVersion, info.Version)
			}
			return printJSON(info)
		},
	}
}
