package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/api"
	"github.com/vibecoder/spoolctl/internal/db"
)

func newFilamentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filament",
		Short: "Manage filament types",
	}
	cmd.AddCommand(
		newFilamentListCmd(),
		newFilamentGetCmd(),
		newFilamentAddCmd(),
		newFilamentEditCmd(),
		newFilamentRmCmd(),
	)
	return cmd
}

func newFilamentListCmd() *cobra.Command {
	var vendorID, material string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List filaments",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.ListFilaments(vendorID, material)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().StringVar(&vendorID, "vendor", "", "Filter by vendor ID")
	cmd.Flags().StringVar(&material, "material", "", "Filter by material (e.g. PLA)")
	return cmd
}

func newFilamentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a filament by ID",
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
			raw, err := c.GetFilament(id)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
}

func newFilamentAddCmd() *cobra.Command {
	var (
		fromDB      string
		vendorID    int
		name        string
		material    string
		density     float64
		diameter    float64
		weight      float64
		spoolWeight float64
		extTemp     int
		bedTemp     int
		colorHex    string
		price       float64
		comment     string
		articleNr   string
		externalID  string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a filament type",
		Long: `Add a filament type. Use --from-db <spoolmandb-id> to auto-fill fields from SpoolmanDB.
Required without --from-db: --density and --diameter.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			var body api.FilamentCreate

			if fromDB != "" {
				spoolDB, err := db.Load(false, flagVerbose)
				if err != nil {
					return fmt.Errorf("loading SpoolmanDB: %w", err)
				}
				ef := spoolDB.Lookup(fromDB)
				if ef == nil {
					return fmt.Errorf("SpoolmanDB record not found: %s", fromDB)
				}
				var vid *int
				if cmd.Flags().Changed("vendor-id") {
					vid = intPtr(vendorID)
				}
				body = db.FilamentToCreate(ef, vid)
				// Allow flag overrides on top of db record
				if cmd.Flags().Changed("name") {
					body.Name = strPtr(name)
				}
				if cmd.Flags().Changed("color-hex") {
					body.ColorHex = strPtr(colorHex)
				}
				if cmd.Flags().Changed("price") {
					body.Price = f64Ptr(price)
				}
			} else {
				if density == 0 || diameter == 0 {
					return fmt.Errorf("--density and --diameter are required without --from-db")
				}
				body = api.FilamentCreate{
					Density:  density,
					Diameter: diameter,
				}
				if cmd.Flags().Changed("vendor-id") {
					body.VendorID = intPtr(vendorID)
				}
				if name != "" {
					body.Name = strPtr(name)
				}
				if material != "" {
					body.Material = strPtr(material)
				}
				if cmd.Flags().Changed("weight") {
					body.Weight = f64Ptr(weight)
				}
				if cmd.Flags().Changed("spool-weight") {
					body.SpoolWeight = f64Ptr(spoolWeight)
				}
				if cmd.Flags().Changed("extruder-temp") {
					body.ExtruderTemp = intPtr(extTemp)
				}
				if cmd.Flags().Changed("bed-temp") {
					body.BedTemp = intPtr(bedTemp)
				}
				if colorHex != "" {
					body.ColorHex = strPtr(colorHex)
				}
				if cmd.Flags().Changed("price") {
					body.Price = f64Ptr(price)
				}
				if comment != "" {
					body.Comment = strPtr(comment)
				}
				if articleNr != "" {
					body.ArticleNumber = strPtr(articleNr)
				}
				if externalID != "" {
					body.ExternalID = strPtr(externalID)
				}
			}

			raw, err := c.CreateFilament(body)
			if err != nil {
				return err
			}
			printRaw(raw)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromDB, "from-db", "", "Auto-fill fields from SpoolmanDB record ID")
	cmd.Flags().IntVar(&vendorID, "vendor-id", 0, "Vendor ID")
	cmd.Flags().StringVar(&name, "name", "", "Filament name")
	cmd.Flags().StringVar(&material, "material", "", "Material (e.g. PLA)")
	cmd.Flags().Float64Var(&density, "density", 0, "Density in g/cm3")
	cmd.Flags().Float64Var(&diameter, "diameter", 0, "Diameter in mm")
	cmd.Flags().Float64Var(&weight, "weight", 0, "Net filament weight in grams")
	cmd.Flags().Float64Var(&spoolWeight, "spool-weight", 0, "Empty spool weight in grams")
	cmd.Flags().IntVar(&extTemp, "extruder-temp", 0, "Extruder temperature in °C")
	cmd.Flags().IntVar(&bedTemp, "bed-temp", 0, "Bed temperature in °C")
	cmd.Flags().StringVar(&colorHex, "color-hex", "", "Color hex code (e.g. FF0000)")
	cmd.Flags().Float64Var(&price, "price", 0, "Price")
	cmd.Flags().StringVar(&comment, "comment", "", "Free text comment")
	cmd.Flags().StringVar(&articleNr, "article-number", "", "Article/EAN number")
	cmd.Flags().StringVar(&externalID, "external-id", "", "External DB ID")
	return cmd
}

func newFilamentEditCmd() *cobra.Command {
	var setFlags []string
	cmd := &cobra.Command{
		Use:   "edit <id> --set key=value",
		Short: "Edit a filament",
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
			body := api.FilamentUpdate{}
			if v, ok := kv["name"]; ok {
				body.Name = strPtr(v)
			}
			if v, ok := kv["material"]; ok {
				body.Material = strPtr(v)
			}
			if v, ok := kv["density"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.Density = &f
				}
			}
			if v, ok := kv["diameter"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.Diameter = &f
				}
			}
			if v, ok := kv["weight"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.Weight = &f
				}
			}
			if v, ok := kv["spool_weight"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.SpoolWeight = &f
				}
			}
			if v, ok := kv["settings_extruder_temp"]; ok {
				if i, err := strconv.Atoi(v); err == nil {
					body.ExtruderTemp = intPtr(i)
				}
			}
			if v, ok := kv["settings_bed_temp"]; ok {
				if i, err := strconv.Atoi(v); err == nil {
					body.BedTemp = intPtr(i)
				}
			}
			if v, ok := kv["color_hex"]; ok {
				body.ColorHex = strPtr(v)
			}
			if v, ok := kv["comment"]; ok {
				body.Comment = strPtr(v)
			}
			if v, ok := kv["price"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					body.Price = &f
				}
			}
			if v, ok := kv["vendor_id"]; ok {
				if i, err := strconv.Atoi(v); err == nil {
					body.VendorID = intPtr(i)
				}
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			raw, err := c.UpdateFilament(id, body)
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

func newFilamentRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a filament",
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
			if err := c.DeleteFilament(id); err != nil {
				return err
			}
			if !flagQuiet {
				fmt.Printf("filament %d deleted\n", id)
			}
			return nil
		},
	}
}
