package portfolio

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "marketview/internal/portfolio/db"
)

// ── toNumeric / fromNumeric ───────────────────────────────────────────────────

func TestToNumeric_Nil(t *testing.T) {
	n := toNumeric(nil)
	if n.Valid {
		t.Error("toNumeric(nil) should return invalid Numeric")
	}
}

func TestToNumeric_NonNil(t *testing.T) {
	f := 100.0
	n := toNumeric(&f)
	if !n.Valid {
		t.Error("toNumeric(&100.0) should return valid Numeric")
	}
}

func TestFromNumeric_Invalid(t *testing.T) {
	n := pgtype.Numeric{} // zero value, Valid = false
	result := fromNumeric(n)
	if result != nil {
		t.Errorf("fromNumeric(invalid) = %v, want nil", *result)
	}
}

func TestToNumeric_RoundTrip_Integer(t *testing.T) {
	f := 42.0
	result := fromNumeric(toNumeric(&f))
	if result == nil {
		t.Fatal("expected non-nil result from round-trip")
	}
	if math.Abs(*result-f) > 0.01 {
		t.Errorf("round-trip: got %v, want %v", *result, f)
	}
}

func TestToNumeric_RoundTrip_Zero(t *testing.T) {
	f := 0.0
	n := toNumeric(&f)
	if !n.Valid {
		t.Error("toNumeric(0.0) should be valid")
	}
}

// ── toDate / fromDate ─────────────────────────────────────────────────────────

func TestToDate_Nil(t *testing.T) {
	d := toDate(nil)
	if d.Valid {
		t.Error("toDate(nil) should return invalid Date")
	}
}

func TestToDate_NonNil(t *testing.T) {
	now := time.Now()
	d := toDate(&now)
	if !d.Valid {
		t.Error("toDate(&time.Now()) should return valid Date")
	}
	if !d.Time.Equal(now) {
		t.Errorf("toDate: got %v, want %v", d.Time, now)
	}
}

func TestFromDate_Invalid(t *testing.T) {
	d := pgtype.Date{} // Valid = false
	result := fromDate(d)
	if result != nil {
		t.Errorf("fromDate(invalid) = %v, want nil", result)
	}
}

func TestFromDate_Valid(t *testing.T) {
	now := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)
	d := pgtype.Date{Time: now, Valid: true}
	result := fromDate(d)
	if result == nil {
		t.Fatal("fromDate(valid) should not return nil")
	}
	if !result.Equal(now) {
		t.Errorf("fromDate: got %v, want %v", result, now)
	}
}

func TestToDate_FromDate_RoundTrip(t *testing.T) {
	original := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	d := toDate(&original)
	result := fromDate(d)
	if result == nil {
		t.Fatal("expected non-nil result from round-trip")
	}
	if !result.Equal(original) {
		t.Errorf("round-trip: got %v, want %v", result, original)
	}
}

// ── fromDB ────────────────────────────────────────────────────────────────────

func TestFromDB_BasicFields(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	meta := json.RawMessage(`{"key":"value"}`)

	row := dbgen.Holding{
		ID:        7,
		AssetType: "stock",
		Name:      "RELIANCE",
		Notes:     "long position",
		Metadata:  meta,
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}

	h := fromDB(row)

	if h.ID != 7 {
		t.Errorf("ID: got %d, want 7", h.ID)
	}
	if h.AssetType != "stock" {
		t.Errorf("AssetType: got %q, want stock", h.AssetType)
	}
	if h.Name != "RELIANCE" {
		t.Errorf("Name: got %q, want RELIANCE", h.Name)
	}
	if h.Notes != "long position" {
		t.Errorf("Notes: got %q, want %q", h.Notes, "long position")
	}
	if string(h.Metadata) != `{"key":"value"}` {
		t.Errorf("Metadata: got %q", string(h.Metadata))
	}
	if !h.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt: got %v, want %v", h.CreatedAt, now)
	}
}

func TestFromDB_NullableFieldsNil(t *testing.T) {
	row := dbgen.Holding{
		ID:        1,
		AssetType: "other",
		Name:      "Gold",
		// Quantity, BuyPrice, CurrentValue, BuyDate all zero/invalid
	}

	h := fromDB(row)

	if h.Quantity != nil {
		t.Errorf("Quantity should be nil for invalid Numeric, got %v", *h.Quantity)
	}
	if h.BuyPrice != nil {
		t.Errorf("BuyPrice should be nil for invalid Numeric, got %v", *h.BuyPrice)
	}
	if h.CurrentValue != nil {
		t.Errorf("CurrentValue should be nil for invalid Numeric, got %v", *h.CurrentValue)
	}
	if h.BuyDate != nil {
		t.Errorf("BuyDate should be nil for invalid Date, got %v", *h.BuyDate)
	}
}

func TestFromDB_AssetTypePreserved(t *testing.T) {
	types := []string{"stock", "fd", "mutual_fund", "gold", "other"}
	for _, at := range types {
		row := dbgen.Holding{AssetType: at, Name: "test"}
		h := fromDB(row)
		if string(h.AssetType) != at {
			t.Errorf("AssetType: got %q, want %q", h.AssetType, at)
		}
	}
}
