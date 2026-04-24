package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check Spoolman server health",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			hc, err := c.GetHealth()
			if err != nil {
				return err
			}
			if !flagQuiet {
				fmt.Printf("status: %s\n", hc.Status)
			}
			return nil
		},
	}
}
