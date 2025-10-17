package tests

import (
	"testing"

	"github.com/mkfoss/foxi"
)

// TestBackendIdentification tests that we can identify which backend is being used
func TestBackendIdentification(t *testing.T) {
	f := foxi.NewFoxi()
	if f == nil {
		t.Fatal("NewFoxi() returned nil")
	}

	backend := f.Backend()
	t.Logf("Using backend: %s", backend.String())

	// Verify backend is one of the expected values
	if backend != foxi.BackendPureGo && backend != foxi.BackendCGO {
		t.Errorf("Unknown backend: %v", backend)
	}
}

// TestInitialState tests that a newly created Foxi instance is in the expected initial state
func TestInitialState(t *testing.T) {
	f := foxi.NewFoxi()
	if f == nil {
		t.Fatal("NewFoxi() returned nil")
	}

	// Should not be active initially
	if f.Active() {
		t.Error("New Foxi instance should not be active")
	}

	// Should return zero values for methods that require an open database
	if f.FieldCount() != 0 {
		t.Errorf("FieldCount() should return 0 for unopened database, got %d", f.FieldCount())
	}

	if f.Position() != 0 {
		t.Errorf("Position() should return 0 for unopened database, got %d", f.Position())
	}

	// EOF/BOF should return true for unopened database
	if !f.EOF() {
		t.Error("EOF() should return true for unopened database")
	}

	if !f.BOF() {
		t.Error("BOF() should return true for unopened database")
	}
}

// TestOpenNonexistentFile tests opening a file that doesn't exist
func TestOpenNonexistentFile(t *testing.T) {
	f := foxi.NewFoxi()
	if f == nil {
		t.Fatal("NewFoxi() returned nil")
	}

	// Try to open a file that doesn't exist
	err := f.Open("nonexistent_file.dbf")
	if err == nil {
		t.Error("Opening nonexistent file should return an error")
		f.Close() // Clean up if somehow it succeeded
	}

	// Should still not be active
	if f.Active() {
		t.Error("Foxi should not be active after failed open")
	}
}

// TestCloseUnopenedDatabase tests closing a database that was never opened
func TestCloseUnopenedDatabase(t *testing.T) {
	f := foxi.NewFoxi()
	if f == nil {
		t.Fatal("NewFoxi() returned nil")
	}

	// Closing an unopened database should not return an error
	err := f.Close()
	if err != nil {
		t.Errorf("Close() on unopened database returned error: %v", err)
	}
}

// TestDoubleOpen tests opening a database when one is already open
func TestDoubleOpen(t *testing.T) {
	f := foxi.NewFoxi()
	if f == nil {
		t.Fatal("NewFoxi() returned nil")
	}

	// This test requires an actual DBF file to work
	// For now, just test the error case with the first open failing
	err1 := f.Open("first_nonexistent.dbf")
	if err1 == nil {
		defer f.Close()
		
		// Try to open another file while one is already open
		err2 := f.Open("second_nonexistent.dbf")
		if err2 == nil {
			t.Error("Opening a second database should return an error")
		}
	}
}

// TestFieldTypeMethods tests the FieldType methods
func TestFieldTypeMethods(t *testing.T) {
	// Test some basic field types
	testCases := []struct {
		ft           foxi.FieldType
		expectedStr  string
		expectedName string
	}{
		{foxi.FTCharacter, "C", "character"},
		{foxi.FTNumeric, "N", "numeric"},
		{foxi.FTDate, "D", "date"},
		{foxi.FTLogical, "L", "logical"},
		{foxi.FTInteger, "I", "integer"},
		{foxi.FTUnknown, "unknown", "unknown"},
	}

	for _, tc := range testCases {
		if tc.ft.String() != tc.expectedStr {
			t.Errorf("FieldType %d String() = %q, expected %q", int(tc.ft), tc.ft.String(), tc.expectedStr)
		}
		if tc.ft.Name() != tc.expectedName {
			t.Errorf("FieldType %d Name() = %q, expected %q", int(tc.ft), tc.ft.Name(), tc.expectedName)
		}
	}
}

// TestCodepageMethods tests the Codepage methods
func TestCodepageMethods(t *testing.T) {
	testCases := []struct {
		cp       foxi.Codepage
		expected string
	}{
		{0x03, "Windows ANSI (1252)"},
		{0x01, "U.S. MS-DOS (437)"},
		{0x02, "International MS-DOS (850)"},
		{0xFF, "Unknown Codepage"}, // Unknown codepage
	}

	for _, tc := range testCases {
		result := tc.cp.String()
		if result != tc.expected {
			t.Errorf("Codepage 0x%02X String() = %q, expected %q", uint8(tc.cp), result, tc.expected)
		}
	}
}

// TestBackendString tests the Backend String() method
func TestBackendString(t *testing.T) {
	testCases := []struct {
		backend  foxi.Backend
		expected string
	}{
		{foxi.BackendPureGo, "Pure Go (gomkfdbf)"},
		{foxi.BackendCGO, "CGO (mkfdbf)"},
		{foxi.Backend(999), "Unknown"}, // Unknown backend
	}

	for _, tc := range testCases {
		result := tc.backend.String()
		if result != tc.expected {
			t.Errorf("Backend %d String() = %q, expected %q", int(tc.backend), result, tc.expected)
		}
	}
}

// TestHeaderZeroValue tests that Header zero value behaves correctly
func TestHeaderZeroValue(t *testing.T) {
	var h foxi.Header

	if h.RecordCount() != 0 {
		t.Errorf("Zero Header RecordCount() = %d, expected 0", h.RecordCount())
	}

	if !h.LastUpdated().IsZero() {
		t.Error("Zero Header LastUpdated() should return zero time")
	}

	if h.HasIndex() {
		t.Error("Zero Header HasIndex() should return false")
	}

	if h.HasFpt() {
		t.Error("Zero Header HasFpt() should return false")
	}

	// Codepage zero value
	if h.Codepage().String() == "" {
		t.Error("Zero Header Codepage() should have a string representation")
	}
}

// BenchmarkNewFoxi benchmarks the creation of new Foxi instances
func BenchmarkNewFoxi(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f := foxi.NewFoxi()
		if f == nil {
			b.Fatal("NewFoxi() returned nil")
		}
	}
}

// BenchmarkBackendString benchmarks the Backend.String() method
func BenchmarkBackendString(b *testing.B) {
	f := foxi.NewFoxi()
	backend := f.Backend()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = backend.String()
	}
}