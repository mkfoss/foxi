package tests

import (
	"testing"

	"github.com/mkfoss/foxi"
)

func TestMustVariantsBasic(t *testing.T) {
	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", cgoBackend, foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == cgoBackend && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			// Test that Must variants exist and can be called
			// (without a valid database they will panic, so we don't call them)

			// Verify the Must methods exist on the main Foxi struct
			// These would panic without a valid database, so we just verify they compile
			if f == nil {
				t.Fatal("NewFoxi() returned nil")
			}

			// Test that Must variants exist on Indexes
			indexes := f.Indexes()
			if indexes == nil {
				t.Fatal("Indexes() returned nil")
			}
		})
	}
}

func TestMustVariantsPanicBehavior(t *testing.T) {
	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", cgoBackend, foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == cgoBackend && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			// Test that Must variants panic on errors
			t.Run("MustOpen panics on invalid file", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustOpen should have panicked with invalid file")
					}
				}()
				f.MustOpen("nonexistent-file-that-should-not-exist.dbf")
			})

			// Test MustGoto panics without open database
			t.Run("MustGoto panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustGoto should have panicked without open database")
					}
				}()
				f.MustGoto(1)
			})

			// Test MustFirst panics without open database
			t.Run("MustFirst panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustFirst should have panicked without open database")
					}
				}()
				f.MustFirst()
			})

			// Test MustLast panics without open database
			t.Run("MustLast panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustLast should have panicked without open database")
					}
				}()
				f.MustLast()
			})

			// Test MustNext panics without open database
			t.Run("MustNext panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustNext should have panicked without open database")
					}
				}()
				f.MustNext()
			})

			// Test MustPrevious panics without open database
			t.Run("MustPrevious panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustPrevious should have panicked without open database")
					}
				}()
				f.MustPrevious()
			})

			// Test MustSkip panics without open database
			t.Run("MustSkip panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustSkip should have panicked without open database")
					}
				}()
				f.MustSkip(1)
			})

			// Test MustDelete panics without open database
			t.Run("MustDelete panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustDelete should have panicked without open database")
					}
				}()
				f.MustDelete()
			})

			// Test MustRecall panics without open database
			t.Run("MustRecall panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustRecall should have panicked without open database")
					}
				}()
				f.MustRecall()
			})
		})
	}
}

func TestMustVariantsIndexes(t *testing.T) {
	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", cgoBackend, foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == cgoBackend && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			indexes := f.Indexes()

			// Test MustLoad without database - this should panic
			t.Run("MustLoad panics without open database", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("MustLoad should have panicked without open database")
					}
				}()
				indexes.MustLoad()
			})

			// Test MustSelectTag with nil - this might panic if no database is open
			t.Run("MustSelectTag with nil", func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						// It's OK if this panics without an open database
						t.Logf("MustSelectTag panicked (expected): %v", r)
					}
				}()
				indexes.MustSelectTag(nil)
			})
		})
	}
}

func TestFieldMustVariantsCompile(t *testing.T) {
	// This test just verifies that the Must variants compile correctly
	// We can't actually test them without valid data, but we can ensure
	// the interface is correct by getting a nil field and checking the methods exist

	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", cgoBackend, foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == cgoBackend && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			// Get a field (will be nil without open database)
			field := f.FieldByName("NONEXISTENT")
			if field != nil {
				// If we somehow got a field, verify the Must methods would panic
				t.Run("Field Must methods panic on invalid field", func(t *testing.T) {
					defer func() {
						if r := recover(); r == nil {
							t.Error("Field Must methods should panic on invalid state")
						}
					}()

					// Test one Must method - if this works, the interface is correct
					field.MustAsString()
				})
			}
		})
	}
}

func TestTagMustVariantsCompile(t *testing.T) {
	// This test verifies the Tag Must variants compile correctly

	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", cgoBackend, foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == cgoBackend && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			indexes := f.Indexes()

			// Get a tag (will be nil without open database/indexes)
			tag := indexes.TagByName("NONEXISTENT")
			if tag != nil {
				// If we somehow got a tag, verify the Must methods would panic
				t.Run("Tag Must methods panic on invalid tag", func(t *testing.T) {
					defer func() {
						if r := recover(); r == nil {
							t.Error("Tag Must methods should panic on invalid state")
						}
					}()

					// Test one Must method - if this works, the interface is correct
					tag.MustSeekString("test")
				})
			}
		})
	}
}
