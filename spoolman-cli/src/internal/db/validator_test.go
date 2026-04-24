package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vibecoder/spoolctl/internal/api"
)

// loadTestDB loads SpoolmanDB from the testdata snapshot.
func loadTestDB(t *testing.T) *DB {
	t.Helper()
	// Walk up from package dir to find testdata
	dir := findTestdata(t)
	fb, err := os.ReadFile(filepath.Join(dir, "spoolmandb-snapshot", "filaments.json"))
	if err != nil {
		t.Skipf("testdata not found: %v", err)
	}
	mb, err := os.ReadFile(filepath.Join(dir, "spoolmandb-snapshot", "materials.json"))
	if err != nil {
		t.Skipf("testdata not found: %v", err)
	}
	db, err := parseDB(fb, mb, SourceCache)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func findTestdata(t *testing.T) string {
	t.Helper()
	// Try relative paths from the module root
	candidates := []string{
		"../../../testdata",
		"../../testdata",
		"../../../../testdata",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "spoolmandb-snapshot")); err == nil {
			return c
		}
	}
	t.Skip("testdata/spoolmandb-snapshot not found")
	return ""
}

func TestNormalizeMaterial(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"PLA", "PLA"},
		{"PLA +", "PLA+"},
		{"PETG ", "PETG"},
		{" ABS", "ABS"},
		{"TPU+", "TPU+"},
		{"PLA+ ", "PLA+"},
	}
	for _, tc := range cases {
		got := normalizeMaterial(tc.input)
		if got != tc.want {
			t.Errorf("normalizeMaterial(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestValidateOK(t *testing.T) {
	testDB := loadTestDB(t)

	spec := &SpecFile{
		Material: "PLA",
		Density:  1.24,
		Diameter: 1.75,
	}
	report := Validate(testDB, spec, "test.json", false)
	if report.Status == StatusError {
		b, _ := json.MarshalIndent(report, "", "  ")
		t.Fatalf("expected ok/warn, got error:\n%s", b)
	}
}

func TestValidateDensityOutOfRange(t *testing.T) {
	testDB := loadTestDB(t)

	spec := &SpecFile{
		Material: "PLA",
		Density:  2.5, // way off
		Diameter: 1.75,
	}
	report := Validate(testDB, spec, "test.json", false)
	if report.Status != StatusWarn {
		t.Errorf("expected warn, got %s", report.Status)
	}
	found := false
	for _, w := range report.Warnings {
		if w.Field == "density" {
			found = true
		}
	}
	if !found {
		t.Error("expected density warning")
	}
}

func TestValidateStrictMode(t *testing.T) {
	testDB := loadTestDB(t)

	spec := &SpecFile{
		Material: "PLA",
		Density:  2.5,
		Diameter: 1.75,
	}
	report := Validate(testDB, spec, "test.json", true)
	if report.Status != StatusError {
		t.Errorf("strict mode: expected error, got %s", report.Status)
	}
}

func TestValidateUnknownMaterial(t *testing.T) {
	testDB := loadTestDB(t)

	spec := &SpecFile{
		Material: "NOTAREAL",
		Density:  1.24,
		Diameter: 1.75,
	}
	report := Validate(testDB, spec, "test.json", false)
	found := false
	for _, w := range report.Warnings {
		if w.Field == "material" {
			found = true
		}
	}
	if !found {
		t.Error("expected unknown material warning")
	}
}

func TestValidateEnumErrors(t *testing.T) {
	testDB := loadTestDB(t)

	spec := &SpecFile{
		Material: "PLA",
		Density:  1.24,
		Diameter: 1.75,
		Finish:   "velvet", // invalid
		Pattern:  "zebra",  // invalid
	}
	report := Validate(testDB, spec, "test.json", false)
	if report.Status != StatusError {
		t.Errorf("expected error for invalid enums, got %s", report.Status)
	}
	fields := map[string]bool{}
	for _, e := range report.Errors {
		fields[e.Field] = true
	}
	if !fields["finish"] {
		t.Error("expected finish error")
	}
	if !fields["pattern"] {
		t.Error("expected pattern error")
	}
}

func TestAutoCorrection(t *testing.T) {
	testDB := loadTestDB(t)

	spec := &SpecFile{
		Material: "PLA +",
		Density:  1.24,
		Diameter: 1.75,
	}
	report := Validate(testDB, spec, "test.json", false)
	if len(report.AutoCorrections) == 0 {
		t.Error("expected auto-correction for 'PLA +'")
	}
	if report.AutoCorrections[0].To != "PLA+" {
		t.Errorf("expected correction to 'PLA+', got %v", report.AutoCorrections[0].To)
	}
}

func TestHardMatchByExternalID(t *testing.T) {
	testDB := loadTestDB(t)
	if len(testDB.Filaments) == 0 {
		t.Skip("no filaments in test DB")
	}

	// Pick first filament from DB
	f := testDB.Filaments[0]
	spec := &SpecFile{
		ExternalID: f.ID,
		Material:   f.Material,
		Density:    f.Density,
		Diameter:   f.Diameter,
	}
	report := Validate(testDB, spec, "test.json", false)
	if report.SuggestedDBID != f.ID {
		t.Errorf("expected suggested_db_id=%q, got %q", f.ID, report.SuggestedDBID)
	}
	if report.MatchConfidence != "high" {
		t.Errorf("expected high confidence, got %s", report.MatchConfidence)
	}
}

func TestLoadSpec(t *testing.T) {
	// Write a temp JSON spec
	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.json")
	spec := `{"material":"PLA","density":1.24,"diameter":1.75}`
	os.WriteFile(specPath, []byte(spec), 0644)

	loaded, err := LoadSpec(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Material != "PLA" || loaded.Density != 1.24 || loaded.Diameter != 1.75 {
		t.Errorf("unexpected spec: %+v", loaded)
	}
}

func TestLoadSpecTOML(t *testing.T) {
	tmp := t.TempDir()
	specPath := filepath.Join(tmp, "spec.toml")
	spec := `material = "PETG"
density = 1.27
diameter = 1.75
`
	os.WriteFile(specPath, []byte(spec), 0644)

	loaded, err := LoadSpec(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Material != "PETG" || loaded.Density != 1.27 {
		t.Errorf("unexpected spec: %+v", loaded)
	}
}

func TestFilterFilaments(t *testing.T) {
	testDB := &DB{
		Filaments: []api.ExternalFilament{
			{ID: "a", Manufacturer: "Bambu Lab", Material: "PLA", Diameter: 1.75},
			{ID: "b", Manufacturer: "Bambu Lab", Material: "PETG", Diameter: 1.75},
			{ID: "c", Manufacturer: "Polymaker", Material: "PLA", Diameter: 1.75},
			{ID: "d", Manufacturer: "Polymaker", Material: "PLA", Diameter: 2.85},
		},
	}

	cases := []struct {
		mfr, mat  string
		dia       float64
		wantCount int
		wantIDs   []string
	}{
		{"", "PLA", 0, 3, []string{"a", "c", "d"}},
		{"Bambu Lab", "", 0, 2, []string{"a", "b"}},
		{"", "PLA", 1.75, 2, []string{"a", "c"}},
		{"Polymaker", "PLA", 2.85, 1, []string{"d"}},
	}

	for _, tc := range cases {
		got := testDB.FilterFilaments(tc.mfr, tc.mat, tc.dia)
		if len(got) != tc.wantCount {
			t.Errorf("FilterFilaments(%q,%q,%.2f): got %d results, want %d",
				tc.mfr, tc.mat, tc.dia, len(got), tc.wantCount)
		}
	}
}

func TestLookup(t *testing.T) {
	testDB := &DB{
		Filaments: []api.ExternalFilament{
			{ID: "bambu-lab_pla-basic_black_1000_175_n", Material: "PLA"},
		},
	}
	f := testDB.Lookup("bambu-lab_pla-basic_black_1000_175_n")
	if f == nil {
		t.Error("expected to find filament")
	}
	if testDB.Lookup("nonexistent") != nil {
		t.Error("expected nil for nonexistent id")
	}
}

func TestFilamentToCreate(t *testing.T) {
	sw := 250.0
	et := 220
	bt := 35
	ef := &api.ExternalFilament{
		ID:           "bambu-lab_tpu_black_1000_175_n",
		Manufacturer: "Bambu Lab",
		Name:         "TPU Black",
		Material:     "TPU",
		Density:      1.22,
		Weight:       1000,
		SpoolWeight:  &sw,
		Diameter:     1.75,
		ExtruderTemp: &et,
		BedTemp:      &bt,
	}
	vid := 1
	fc := FilamentToCreate(ef, &vid)
	if fc.VendorID == nil || *fc.VendorID != 1 {
		t.Errorf("expected vendor_id=1, got %v", fc.VendorID)
	}
	if fc.Density != 1.22 {
		t.Errorf("expected density 1.22, got %f", fc.Density)
	}
	if fc.ExternalID == nil || *fc.ExternalID != ef.ID {
		t.Errorf("expected external_id=%s", ef.ID)
	}
	if fc.ExtruderTemp == nil || *fc.ExtruderTemp != 220 {
		t.Errorf("expected extruder_temp=220")
	}
}
