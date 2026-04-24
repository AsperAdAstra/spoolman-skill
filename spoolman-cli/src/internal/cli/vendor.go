package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/api"
)

func newVendorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vendor",
		Short: "Manage vendors",
	}
	cmd.AddCommand(
		newVendorListCmd(),
		newVendorGetCmd(),
		newVendorAddCmd(),
		newVendorEditCmd(),
		newVendorRmCmd(),
	)
	return cmd
}

func newVendorListCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List vendors",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.ListVendors(name)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Filter by name")
	return cmd
}

func newVendorGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a vendor by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %s", args[0])
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.GetVendor(id)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
}

func newVendorAddCmd() *cobra.Command {
	var (
		name        string
		comment     string
		spoolWeight float64
		externalID  string
		extra       []string
	)
	cmd := &cobra.Command{
		Use:   "add --name <name>",
		Short: "Add a vendor",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			body := api.VendorCreate{Name: name}
			if cmd.Flags().Changed("comment") {
				body.Comment = strPtr(comment)
			}
			if cmd.Flags().Changed("spool-weight") {
				body.EmptySpoolWeight = &spoolWeight
			}
			if cmd.Flags().Changed("external-id") {
				body.ExternalID = strPtr(externalID)
			}
			if len(extra) > 0 {
				body.Extra = parseKV(extra)
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.CreateVendor(body)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Vendor name (required)")
	cmd.Flags().StringVar(&comment, "comment", "", "Free text comment")
	cmd.Flags().Float64Var(&spoolWeight, "spool-weight", 0, "Default empty spool weight in grams")
	cmd.Flags().StringVar(&externalID, "external-id", "", "External DB ID")
	cmd.Flags().StringArrayVar(&extra, "extra", nil, "Extra fields as key=value")
	return cmd
}

func newVendorEditCmd() *cobra.Command {
	var setFlags []string
	cmd := &cobra.Command{
		Use:   "edit <id> --set key=value",
		Short: "Edit a vendor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %s", args[0])
			}
			if len(setFlags) == 0 {
				return fmt.Errorf("at least one --set key=value required")
			}
			kv := parseKV(setFlags)
			body := api.VendorUpdate{}
			if v, ok := kv["name"]; ok {
				body.Name = strPtr(v)
			}
			if v, ok := kv["comment"]; ok {
				body.Comment = strPtr(v)
			}
			if v, ok := kv["external_id"]; ok {
				body.ExternalID = strPtr(v)
			}
			if v, ok := kv["empty_spool_weight"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.EmptySpoolWeight = &f
				}
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.UpdateVendor(id, body)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&setFlags, "set", nil, "Fields to set as key=value")
	return cmd
}

func newVendorRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a vendor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %s", args[0])
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			if err := c.DeleteVendor(id); err != nil {
				return err
			}
			if !flagQuiet {
				fmt.Printf("vendor %d deleted\n", id)
			}
			return nil
		},
	}
}

// parseKV parses ["key=value", ...] into a map.
func parseKV(pairs []string) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}
