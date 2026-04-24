package db

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/vibecoder/spoolctl/internal/api"
)

// ValidStatus classifies the overall validation result.
type ValidStatus string

const (
	StatusOK    ValidStatus = "ok"
	StatusWarn  ValidStatus = "warn"
	StatusError ValidStatus = "error"
)

// FieldMatch records a successful field match.
type FieldMatch struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	DBValue interface{} `json:"db_value"`
}

// FieldWarning records an out-of-range or advisory issue.
type FieldWarning struct {
	Field         string      `json:"field"`
	Value         interface{} `json:"value"`
	ExpectedRange interface{} `json:"expected_range,omitempty"`
	Material      string      `json:"material,omitempty"`
	Message       string      `json:"message,omitempty"`
}

// FieldError records a hard mismatch.
type FieldError struct {
	Field    string      `json:"field"`
	Value    interface{} `json:"value"`
	Expected interface{} `json:"expected"`
	Message  string      `json:"message,omitempty"`
}

// AutoCorrection records a safe normalization applied.
type AutoCorrection struct {
	Field string      `json:"field"`
	From  interface{} `json:"from"`
	To    interface{} `json:"to"`
}

// ValidationReport is the output of db validate.
type ValidationReport struct {
	Input               string           `json:"input"`
	Status              ValidStatus      `json:"status"`
	Matches             []FieldMatch     `json:"matches"`
	Warnings            []FieldWarning   `json:"warnings"`
	Errors              []FieldError     `json:"errors"`
	SuggestedDBID       string           `json:"suggested_db_id,omitempty"`
	MatchConfidence     string           `json:"match_confidence,omitempty"`
	AutoCorrections     []AutoCorrection `json:"auto_corrections"`
	RequiresConfirmation bool            `json:"requires_confirmation"`
}

// SpecFile is the input spec for validation (subset of fields we check).
type SpecFile struct {
	// SpoolmanDB id (optional shortcut to hard-match)
	ExternalID string `json:"external_id" toml:"external_id"`

	Manufacturer string  `json:"manufacturer" toml:"manufacturer"`
	Name         string  `json:"name"         toml:"name"`
	Material     string  `json:"material"     toml:"material"`
	Density      float64 `json:"density"      toml:"density"`
	Diameter     float64 `json:"diameter"     toml:"diameter"`
	ExtruderTemp int     `json:"extruder_temp" toml:"extruder_temp"`
	BedTemp      int     `json:"bed_temp"     toml:"bed_temp"`
	SpoolWeight  float64 `json:"spool_weight" toml:"spool_weight"`
	Weight       float64 `json:"weight"       toml:"weight"`
	Finish       string  `json:"finish"       toml:"finish"`
	Pattern      string  `json:"pattern"      toml:"pattern"`
	SpoolType    string  `json:"spool_type"   toml:"spool_type"`
}

var validFinish = map[string]bool{"matte": true, "glossy": true}
var validPattern = map[string]bool{"marble": true, "sparkle": true}
var validSpoolType = map[string]bool{"plastic": true, "cardboard": true, "metal": true}
var validMultiColorDir = map[string]bool{"coaxial": true, "longitudinal": true}

// LoadSpec loads a SpecFile from a TOML or JSON file.
func LoadSpec(path string) (*SpecFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec SpecFile
	if strings.HasSuffix(path, ".toml") {
		if err := toml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing TOML: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}
	}
	return &spec, nil
}

// Validate runs the three-pass validator against db and returns a report.
func Validate(db *DB, spec *SpecFile, inputPath string, strict bool) *ValidationReport {
	r := &ValidationReport{
		Input:           inputPath,
		Status:          StatusOK,
		Matches:         []FieldMatch{},
		Warnings:        []FieldWarning{},
		Errors:          []FieldError{},
		AutoCorrections: []AutoCorrection{},
	}

	// --- Pass 0: safe auto-corrections ---
	spec = applyAutoCorrections(spec, r)

	// --- Pass 1: hard match if external_id or derivable ---
	var dbFilament *api.ExternalFilament
	if spec.ExternalID != "" {
		dbFilament = db.Lookup(spec.ExternalID)
		if dbFilament != nil {
			r.SuggestedDBID = dbFilament.ID
			r.MatchConfidence = "high"
			hardMatch(spec, dbFilament, r)
		}
	}

	// --- If no hard match, try to find best near-match ---
	if dbFilament == nil && spec.Manufacturer != "" && spec.Name != "" {
		candidates := findCandidates(db, spec)
		if len(candidates) == 1 {
			dbFilament = &candidates[0]
			r.SuggestedDBID = dbFilament.ID
			r.MatchConfidence = "medium"
		} else if len(candidates) > 1 {
			r.MatchConfidence = "low"
			r.Warnings = append(r.Warnings, FieldWarning{
				Field:   "db_match",
				Message: fmt.Sprintf("%d near-matches found; cannot auto-select", len(candidates)),
			})
		}
	}

	// --- Pass 2: material sanity ---
	if spec.Material != "" {
		mat := db.FindMaterial(spec.Material)
		if mat == nil {
			r.Warnings = append(r.Warnings, FieldWarning{
				Field:   "material",
				Value:   spec.Material,
				Message: "material not found in SpoolmanDB",
			})
		} else {
			checkMaterialSanity(spec, mat, r)
		}
	}

	// --- Pass 3: enum sanity ---
	checkEnums(spec, r)

	// --- Determine requires_confirmation ---
	r.RequiresConfirmation = requiresConfirmation(r)

	// --- Compute status ---
	if len(r.Errors) > 0 {
		r.Status = StatusError
	} else if len(r.Warnings) > 0 {
		r.Status = StatusWarn
	}

	if strict && r.Status != StatusOK {
		// mark errors for any warnings too when strict
		for _, w := range r.Warnings {
			r.Errors = append(r.Errors, FieldError{
				Field:   w.Field,
				Value:   w.Value,
				Message: w.Message,
			})
		}
		if len(r.Errors) > 0 {
			r.Status = StatusError
		}
	}

	return r
}

func applyAutoCorrections(spec *SpecFile, r *ValidationReport) *SpecFile {
	out := *spec

	// Normalize material spacing variants
	normalized := normalizeMaterial(spec.Material)
	if normalized != spec.Material {
		r.AutoCorrections = append(r.AutoCorrections, AutoCorrection{
			Field: "material",
			From:  spec.Material,
			To:    normalized,
		})
		out.Material = normalized
	}

	// Normalize manufacturer spacing
	if spec.Manufacturer != "" {
		clean := strings.TrimSpace(spec.Manufacturer)
		if clean != spec.Manufacturer {
			r.AutoCorrections = append(r.AutoCorrections, AutoCorrection{
				Field: "manufacturer",
				From:  spec.Manufacturer,
				To:    clean,
			})
			out.Manufacturer = clean
		}
	}

	return &out
}

// normalizeMaterial handles variants like "PLA +" -> "PLA+", "PETG " -> "PETG".
func normalizeMaterial(m string) string {
	m = strings.TrimSpace(m)
	// Collapse "PLA +" -> "PLA+"
	m = strings.ReplaceAll(m, " +", "+")
	m = strings.ReplaceAll(m, "+ ", "+")
	return m
}

func hardMatch(spec *SpecFile, db *api.ExternalFilament, r *ValidationReport) {
	checkField := func(field string, specVal, dbVal interface{}, tolerance float64) {
		switch s := specVal.(type) {
		case float64:
			dv := dbVal.(float64)
			if tolerance > 0 {
				if math.Abs(s-dv)/math.Max(dv, 0.001) > tolerance {
					r.Errors = append(r.Errors, FieldError{
						Field:    field,
						Value:    s,
						Expected: dv,
					})
					return
				}
			} else if s != dv {
				r.Errors = append(r.Errors, FieldError{
					Field:    field,
					Value:    s,
					Expected: dv,
				})
				return
			}
			r.Matches = append(r.Matches, FieldMatch{Field: field, Value: s, DBValue: dv})
		case string:
			dv := dbVal.(string)
			if !strings.EqualFold(s, dv) {
				r.Errors = append(r.Errors, FieldError{
					Field:    field,
					Value:    s,
					Expected: dv,
				})
				return
			}
			r.Matches = append(r.Matches, FieldMatch{Field: field, Value: s, DBValue: dv})
		}
	}

	checkField("diameter", spec.Diameter, db.Diameter, 0.01)
	checkField("material", spec.Material, db.Material, 0)
	checkField("density", spec.Density, db.Density, 0.05)
	if spec.Weight > 0 {
		checkField("weight", spec.Weight, db.Weight, 0.05)
	}
	if spec.SpoolWeight > 0 && db.SpoolWeight != nil {
		checkField("spool_weight", spec.SpoolWeight, *db.SpoolWeight, 0.05)
	}
	if spec.ExtruderTemp > 0 && db.ExtruderTemp != nil {
		// temps are advisory in hard-match pass
		if abs(float64(spec.ExtruderTemp-*db.ExtruderTemp)) > 15 {
			r.Warnings = append(r.Warnings, FieldWarning{
				Field:   "extruder_temp",
				Value:   spec.ExtruderTemp,
				Message: fmt.Sprintf("differs from SpoolmanDB value %d by more than 15°C", *db.ExtruderTemp),
			})
		} else {
			r.Matches = append(r.Matches, FieldMatch{
				Field:   "extruder_temp",
				Value:   spec.ExtruderTemp,
				DBValue: *db.ExtruderTemp,
			})
		}
	}
	if spec.BedTemp > 0 && db.BedTemp != nil {
		if abs(float64(spec.BedTemp-*db.BedTemp)) > 15 {
			r.Warnings = append(r.Warnings, FieldWarning{
				Field:   "bed_temp",
				Value:   spec.BedTemp,
				Message: fmt.Sprintf("differs from SpoolmanDB value %d by more than 15°C", *db.BedTemp),
			})
		} else {
			r.Matches = append(r.Matches, FieldMatch{
				Field:   "bed_temp",
				Value:   spec.BedTemp,
				DBValue: *db.BedTemp,
			})
		}
	}
}

func checkMaterialSanity(spec *SpecFile, mat *api.ExternalMaterial, r *ValidationReport) {
	// Density: ±10%
	if spec.Density > 0 {
		if diff := math.Abs(spec.Density-mat.Density) / mat.Density; diff > 0.10 {
			r.Warnings = append(r.Warnings, FieldWarning{
				Field:    "density",
				Value:    spec.Density,
				Material: spec.Material,
				Message:  fmt.Sprintf("%.3f is >10%% from material default %.3f", spec.Density, mat.Density),
			})
		} else {
			r.Matches = append(r.Matches, FieldMatch{
				Field:   "density",
				Value:   spec.Density,
				DBValue: mat.Density,
			})
		}
	}
	// Extruder temp: ±15°C
	if spec.ExtruderTemp > 0 && mat.ExtruderTemp != nil {
		if d := abs(float64(spec.ExtruderTemp - *mat.ExtruderTemp)); d > 15 {
			r.Warnings = append(r.Warnings, FieldWarning{
				Field:         "extruder_temp",
				Value:         spec.ExtruderTemp,
				ExpectedRange: [2]int{*mat.ExtruderTemp - 15, *mat.ExtruderTemp + 15},
				Material:      spec.Material,
			})
		}
	}
	// Bed temp: ±15°C
	if spec.BedTemp > 0 && mat.BedTemp != nil {
		if d := abs(float64(spec.BedTemp - *mat.BedTemp)); d > 15 {
			r.Warnings = append(r.Warnings, FieldWarning{
				Field:         "bed_temp",
				Value:         spec.BedTemp,
				ExpectedRange: [2]int{*mat.BedTemp - 15, *mat.BedTemp + 15},
				Material:      spec.Material,
			})
		}
	}
}

func checkEnums(spec *SpecFile, r *ValidationReport) {
	if spec.Finish != "" && !validFinish[strings.ToLower(spec.Finish)] {
		r.Errors = append(r.Errors, FieldError{
			Field:    "finish",
			Value:    spec.Finish,
			Expected: []string{"matte", "glossy"},
		})
	}
	if spec.Pattern != "" && !validPattern[strings.ToLower(spec.Pattern)] {
		r.Errors = append(r.Errors, FieldError{
			Field:    "pattern",
			Value:    spec.Pattern,
			Expected: []string{"marble", "sparkle"},
		})
	}
	if spec.SpoolType != "" && !validSpoolType[strings.ToLower(spec.SpoolType)] {
		r.Errors = append(r.Errors, FieldError{
			Field:    "spool_type",
			Value:    spec.SpoolType,
			Expected: []string{"plastic", "cardboard", "metal"},
		})
	}
}

func requiresConfirmation(r *ValidationReport) bool {
	if len(r.Errors) > 0 {
		return true
	}
	if r.MatchConfidence == "low" || r.MatchConfidence == "" && r.SuggestedDBID == "" {
		return false // no match to confirm against
	}
	return false
}

func findCandidates(db *DB, spec *SpecFile) []api.ExternalFilament {
	var out []api.ExternalFilament
	mfr := strings.ToLower(spec.Manufacturer)
	name := strings.ToLower(spec.Name)
	for _, f := range db.Filaments {
		if mfr != "" && !strings.Contains(strings.ToLower(f.Manufacturer), mfr) {
			continue
		}
		if name != "" && !strings.Contains(strings.ToLower(f.Name), name) {
			continue
		}
		if spec.Material != "" && !strings.EqualFold(f.Material, spec.Material) {
			continue
		}
		if spec.Diameter > 0 && abs(f.Diameter-spec.Diameter) > 0.01 {
			continue
		}
		out = append(out, f)
	}
	return out
}
