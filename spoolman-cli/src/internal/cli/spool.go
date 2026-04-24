package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/api"
)

func newSpoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spool",
		Short: "Manage spools",
	}
	cmd.AddCommand(
		newSpoolListCmd(),
		newSpoolGetCmd(),
		newSpoolAddCmd(),
		newSpoolEditCmd(),
		newSpoolRmCmd(),
		newSpoolUseCmd(),
		newSpoolMeasureCmd(),
	)
	return cmd
}

func newSpoolListCmd() *cobra.Command {
	var filamentID string
	var archived bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List spools",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.ListSpools(filamentID, archived)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().StringVar(&filamentID, "filament", "", "Filter by filament ID")
	cmd.Flags().BoolVar(&archived, "archived", false, "Include archived spools")
	return cmd
}

func newSpoolGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a spool by ID",
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
			raw, err := c.GetSpool(id)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
}

func newSpoolAddCmd() *cobra.Command {
	var (
		filamentID    int
		initialWeight float64
		spoolWeight   float64
		remainWeight  float64
		usedWeight    float64
		price         float64
		location      string
		lotNr         string
		comment       string
		archived      bool
	)
	cmd := &cobra.Command{
		Use:   "add --filament-id <id>",
		Short: "Add a spool",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("filament-id") {
				return fmt.Errorf("--filament-id is required")
			}
			body := api.SpoolCreate{FilamentID: filamentID}
			if cmd.Flags().Changed("initial-weight") {
				body.InitialWeight = f64Ptr(initialWeight)
			}
			if cmd.Flags().Changed("spool-weight") {
				body.SpoolWeight = f64Ptr(spoolWeight)
			}
			if cmd.Flags().Changed("remaining-weight") {
				body.RemainingWeight = f64Ptr(remainWeight)
			}
			if cmd.Flags().Changed("used-weight") {
				body.UsedWeight = f64Ptr(usedWeight)
			}
			if cmd.Flags().Changed("price") {
				body.Price = f64Ptr(price)
			}
			if location != "" {
				body.Location = strPtr(location)
			}
			if lotNr != "" {
				body.LotNr = strPtr(lotNr)
			}
			if comment != "" {
				body.Comment = strPtr(comment)
			}
			body.Archived = archived
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.CreateSpool(body)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().IntVar(&filamentID, "filament-id", 0, "Filament type ID (required)")
	cmd.Flags().Float64Var(&initialWeight, "initial-weight", 0, "Initial net filament weight in grams")
	cmd.Flags().Float64Var(&spoolWeight, "spool-weight", 0, "Empty spool (tare) weight in grams")
	cmd.Flags().Float64Var(&remainWeight, "remaining-weight", 0, "Remaining filament weight in grams")
	cmd.Flags().Float64Var(&usedWeight, "used-weight", 0, "Used filament weight in grams")
	cmd.Flags().Float64Var(&price, "price", 0, "Spool price")
	cmd.Flags().StringVar(&location, "location", "", "Storage location")
	cmd.Flags().StringVar(&lotNr, "lot-nr", "", "Lot/batch number")
	cmd.Flags().StringVar(&comment, "comment", "", "Free text comment")
	cmd.Flags().BoolVar(&archived, "archived", false, "Mark spool as archived")
	return cmd
}

func newSpoolEditCmd() *cobra.Command {
	var setFlags []string
	cmd := &cobra.Command{
		Use:   "edit <id> --set key=value",
		Short: "Edit a spool",
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
			body := api.SpoolUpdate{}
			if v, ok := kv["filament_id"]; ok {
				if i, err := strconv.Atoi(v); err == nil {
					body.FilamentID = intPtr(i)
				}
			}
			if v, ok := kv["location"]; ok {
				body.Location = strPtr(v)
			}
			if v, ok := kv["comment"]; ok {
				body.Comment = strPtr(v)
			}
			if v, ok := kv["lot_nr"]; ok {
				body.LotNr = strPtr(v)
			}
			if v, ok := kv["price"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.Price = &f
				}
			}
			if v, ok := kv["initial_weight"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.InitialWeight = &f
				}
			}
			if v, ok := kv["spool_weight"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.SpoolWeight = &f
				}
			}
			if v, ok := kv["remaining_weight"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.RemainingWeight = &f
				}
			}
			if v, ok := kv["used_weight"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.UsedWeight = &f
				}
			}
			if v, ok := kv["archived"]; ok {
				b := v == "true" || v == "1"
				body.Archived = boolPtr(b)
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.UpdateSpool(id, body)
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

func newSpoolRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a spool",
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
			if err := c.DeleteSpool(id); err != nil {
				return err
			}
			if !flagQuiet {
				fmt.Printf("spool %d deleted\n", id)
			}
			return nil
		},
	}
}

func newSpoolUseCmd() *cobra.Command {
	var (
		weight float64
		length float64
		ref    string
	)
	cmd := &cobra.Command{
		Use:   "use <id>",
		Short: "Record filament usage on a spool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %s", args[0])
			}
			if !cmd.Flags().Changed("weight") && !cmd.Flags().Changed("length") {
				return fmt.Errorf("at least one of --weight or --length required")
			}
			body := api.SpoolUse{}
			if cmd.Flags().Changed("weight") {
				body.UseWeight = f64Ptr(weight)
			}
			if cmd.Flags().Changed("length") {
				body.UseLength = f64Ptr(length)
			}
			_ = ref // stored in comment via edit if needed; API doesn't have a ref field
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.UseSpool(id, body)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().Float64Var(&weight, "weight", 0, "Weight of filament used in grams")
	cmd.Flags().Float64Var(&length, "length", 0, "Length of filament used in mm")
	cmd.Flags().StringVar(&ref, "ref", "", "Print job reference (informational)")
	return cmd
}

func newSpoolMeasureCmd() *cobra.Command {
	var weight float64
	cmd := &cobra.Command{
		Use:   "measure <id> --weight <grams>",
		Short: "Set remaining weight by scale measurement (gross weight)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %s", args[0])
			}
			if !cmd.Flags().Changed("weight") {
				return fmt.Errorf("--weight is required")
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.MeasureSpool(id, api.SpoolMeasure{Weight: weight})
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().Float64Var(&weight, "weight", 0, "Current gross weight of spool in grams (required)")
	return cmd
}
