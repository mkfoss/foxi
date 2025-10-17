package tests

import (
	"testing"

	"github.com/mkfoss/foxi"
)

func TestIndexesBasicOperations(t *testing.T) {
	// Test with both backends
	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", "foxicgo", foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == "foxicgo" && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			// Test accessing indexes before database is open
			indexes := f.Indexes()
			if indexes == nil {
				t.Error("Indexes() returned nil before database open")
			}

			// Should not be loaded initially
			if indexes.Loaded() {
				t.Error("Indexes should not be loaded initially")
			}

			// Count should be 0 when not loaded
			if count := indexes.Count(); count != 0 {
				t.Errorf("Expected count 0 when not loaded, got %d", count)
			}

			// List should be empty when not loaded and auto-load should fail (no DB)
			list := indexes.List()
			if list != nil && len(list) > 0 {
				t.Error("Expected empty list when no database open")
			}
		})
	}
}

func TestIndexesLazyLoading(t *testing.T) {
	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", "foxicgo", foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == "foxicgo" && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			// Open test database (if available)
			err := f.Open("test_data/sample.dbf")
			if err != nil {
				// Skip if no test data available
				t.Skip("No test database available for index testing")
			}

			indexes := f.Indexes()
			if indexes == nil {
				t.Fatal("Indexes() returned nil")
			}

			// Initially should not be loaded
			if indexes.Loaded() {
				t.Error("Indexes should not be loaded initially")
			}

			// Force loading
			err = indexes.Load()
			if err != nil {
				t.Logf("Index loading failed: %v (this may be expected if no index files exist)", err)
			}

			// After loading attempt, should be marked as loaded
			if !indexes.Loaded() {
				t.Error("Indexes should be marked as loaded after Load() call")
			}

			// Count should be accessible
			count := indexes.Count()
			t.Logf("Found %d indexes", count)

			// List should be accessible
			list := indexes.List()
			if list == nil {
				t.Error("List() returned nil after loading")
			}
			if len(list) != count {
				t.Errorf("List length %d doesn't match Count %d", len(list), count)
			}
		})
	}
}

func TestIndexAccess(t *testing.T) {
	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", "foxicgo", foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == "foxicgo" && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			err := f.Open("test_data/sample.dbf")
			if err != nil {
				t.Skip("No test database available for index testing")
			}

			indexes := f.Indexes()

			// Test ByIndex with invalid indices
			if idx := indexes.ByIndex(-1); idx != nil {
				t.Error("ByIndex(-1) should return nil")
			}
			if idx := indexes.ByIndex(9999); idx != nil {
				t.Error("ByIndex(9999) should return nil for non-existent index")
			}

			// Test ByName with non-existent name
			if idx := indexes.ByName("NONEXISTENT"); idx != nil {
				t.Error("ByName(NONEXISTENT) should return nil")
			}

			// If we have indexes, test accessing them
			if indexes.Count() > 0 {
				list := indexes.List()
				for i, idx := range list {
					if idx == nil {
						t.Errorf("Index %d is nil", i)
						continue
					}

					// Test Index interface
					name := idx.Name()
					fileName := idx.FileName()
					tagCount := idx.TagCount()
					isOpen := idx.IsOpen()
					isProduction := idx.IsProduction()

					t.Logf("Index %d: Name=%s, File=%s, Tags=%d, Open=%t, Production=%t",
						i, name, fileName, tagCount, isOpen, isProduction)

					// Test accessing by index
					sameIdx := indexes.ByIndex(i)
					if sameIdx != idx {
						t.Errorf("ByIndex(%d) returned different instance", i)
					}

					// Test accessing by name
					if name != "" {
						namedIdx := indexes.ByName(name)
						if namedIdx != idx {
							t.Errorf("ByName(%s) returned different instance", name)
						}
					}

					// Test tags if available
					if tagCount > 0 {
						tags := idx.Tags()
						if len(tags) != tagCount {
							t.Errorf("Tags() returned %d tags, expected %d", len(tags), tagCount)
						}

						for j, tag := range tags {
							if tag == nil {
								t.Errorf("Tag %d is nil", j)
								continue
							}

							testTag(t, tag, j)
						}
					}
				}
			}
		})
	}
}

func TestTagOperations(t *testing.T) {
	testCases := []struct {
		name     string
		backend  string
		expected foxi.Backend
	}{
		{"Pure Go Backend", "", foxi.BackendPureGo},
		{"CGO Backend", "foxicgo", foxi.BackendCGO},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == "foxicgo" && !cgoBuildTagPresent() {
				t.Skip("CGO backend not available in this build")
			}

			f := foxi.NewFoxi()
			defer f.Close()

			err := f.Open("test_data/sample.dbf")
			if err != nil {
				t.Skip("No test database available for index testing")
			}

			indexes := f.Indexes()

			// Test TagByName with non-existent name
			if tag := indexes.TagByName("NONEXISTENT"); tag != nil {
				t.Error("TagByName(NONEXISTENT) should return nil")
			}

			// Test SelectedTag initially (should be nil)
			if tag := indexes.SelectedTag(); tag != nil {
				t.Log("Found initially selected tag:", tag.Name())
			}

			// Test Tags()
			allTags := indexes.Tags()
			t.Logf("Found %d total tags across all indexes", len(allTags))

			// Test tag selection
			if len(allTags) > 0 {
				firstTag := allTags[0]

				// Test selecting a tag
				err := indexes.SelectTag(firstTag)
				if err != nil {
					t.Errorf("Failed to select tag: %v", err)
				} else {
					// Verify it's selected
					selected := indexes.SelectedTag()
					if selected != firstTag {
						t.Error("Selected tag doesn't match what we set")
					}
					if !firstTag.IsSelected() {
						t.Error("Tag should report as selected")
					}
				}

				// Test clearing selection
				err = indexes.SelectTag(nil)
				if err != nil {
					t.Errorf("Failed to clear tag selection: %v", err)
				} else {
					selected := indexes.SelectedTag()
					if selected != nil {
						t.Error("Tag selection should be cleared")
					}
				}
			}
		})
	}
}

func testTag(t *testing.T, tag foxi.Tag, index int) {
	// Test basic properties
	name := tag.Name()
	expr := tag.Expression()
	filter := tag.Filter()
	keyLen := tag.KeyLength()
	unique := tag.IsUnique()
	desc := tag.IsDescending()
	selected := tag.IsSelected()

	t.Logf("Tag %d: Name=%s, Expr=%s, Filter=%s, KeyLen=%d, Unique=%t, Desc=%t, Selected=%t",
		index, name, expr, filter, keyLen, unique, desc, selected)

	// Test seek operations (these may fail if no data or wrong types)
	testSeekOperations(t, tag)

	// Test navigation operations (these may fail if tag is empty)
	testTagNavigation(t, tag)
}

func testSeekOperations(t *testing.T, tag foxi.Tag) {
	// Test seeking with various value types
	testValues := []interface{}{
		"TEST",
		123,
		45.67,
	}

	for _, value := range testValues {
		result, err := tag.Seek(value)
		if err != nil {
			t.Logf("Seek(%v) failed: %v (this may be expected)", value, err)
		} else {
			t.Logf("Seek(%v) result: %s", value, result.String())
		}
	}

	// Test specific type seeks
	result, err := tag.SeekString("TEST")
	if err != nil {
		t.Logf("SeekString failed: %v (this may be expected)", err)
	} else {
		t.Logf("SeekString result: %s", result.String())
	}

	result, err = tag.SeekInt(123)
	if err != nil {
		t.Logf("SeekInt failed: %v (this may be expected)", err)
	} else {
		t.Logf("SeekInt result: %s", result.String())
	}

	result, err = tag.SeekDouble(45.67)
	if err != nil {
		t.Logf("SeekDouble failed: %v (this may be expected)", err)
	} else {
		t.Logf("SeekDouble result: %s", result.String())
	}
}

func testTagNavigation(t *testing.T, tag foxi.Tag) {
	// Test navigation - these operations may fail if tag is empty or not selected
	err := tag.First()
	if err != nil {
		t.Logf("First() failed: %v (this may be expected)", err)
		return // Can't test further navigation if First fails
	}

	// Test state after First
	eof := tag.EOF()
	bof := tag.BOF()
	pos := tag.Position()
	key := tag.CurrentKey()
	recNo := tag.RecordNumber()

	t.Logf("After First(): EOF=%t, BOF=%t, Pos=%f, Key=%s, RecNo=%d",
		eof, bof, pos, key, recNo)

	// Try Next
	err = tag.Next()
	if err != nil {
		t.Logf("Next() failed: %v (this may be expected)", err)
	} else {
		t.Logf("Next() succeeded, new position: %f", tag.Position())
	}

	// Try Last
	err = tag.Last()
	if err != nil {
		t.Logf("Last() failed: %v (this may be expected)", err)
	} else {
		t.Logf("Last() succeeded, position: %f", tag.Position())
	}

	// Try Previous
	err = tag.Previous()
	if err != nil {
		t.Logf("Previous() failed: %v (this may be expected)", err)
	} else {
		t.Logf("Previous() succeeded, position: %f", tag.Position())
	}

	// Try position set
	err = tag.PositionSet(0.5)
	if err != nil {
		t.Logf("PositionSet(0.5) failed: %v (this may be expected)", err)
	} else {
		t.Logf("PositionSet(0.5) succeeded, new position: %f", tag.Position())
	}
}

func TestSeekResultString(t *testing.T) {
	tests := []struct {
		result   foxi.SeekResult
		expected string
	}{
		{foxi.SeekSuccess, "success"},
		{foxi.SeekAfter, "after"},
		{foxi.SeekEOF, "eof"},
		{foxi.SeekResult(999), "unknown"},
	}

	for _, test := range tests {
		if got := test.result.String(); got != test.expected {
			t.Errorf("SeekResult(%d).String() = %s, want %s", int(test.result), got, test.expected)
		}
	}
}

// Helper function to determine if CGO build tag is present
func cgoBuildTagPresent() bool {
	// This is a simple check - in a real implementation you might
	// check build tags more sophisticatedly
	return false // Assume false for testing pure Go backend by default
}
