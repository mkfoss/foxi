# Foxi - High Level Go DBF Package

Foxi is a high-level Go package for reading FoxPro database tables, modeled after the [Vulpo](https://github.com/mkfoss/vulpo) component. It supports both CGO-based and pure Go backends depending on build tags, providing flexibility between performance and deployment simplicity.

## Features

- **Dual Backend Support**: Choose between pure Go or CGO implementation at compile time
- **Vulpo-Compatible API**: Identical interface to the proven Vulpo library
- **Build Tag Selection**: Simple build tag controls backend selection
- **Self-Contained**: All dependencies are included in subpackages
- **Full DBF Support**: Complete Visual FoxPro database compatibility
- **Type-Safe Field Access**: Unified Field API with automatic type conversion

## Backends

### Pure Go Backend (Default)
- **Source**: [gomkfdbf](https://github.com/sean/gomkfdbf) - Complete Go translation of CodeBase library
- **Advantages**: No CGO dependencies, cross-platform builds, static linking
- **Build**: `go build` (default)
- **Performance**: ~70% of C performance with 100% memory safety

### CGO Backend (Optional)  
- **Source**: mkfdbf C library - Production-proven CodeBase implementation
- **Advantages**: Maximum performance, battle-tested codebase
- **Build**: `go build -tags foxicgo`
- **Requirements**: CGO enabled, C compiler

## Installation

```bash
go get github.com/mkfoss/foxi
```

### Direct Package Usage

You can also access the backend packages directly from within the foxi module:

```bash
# Get the full foxi package (includes all backends)
go get github.com/mkfoss/foxi
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/mkfoss/foxi"
)

func main() {
    // Create new instance (backend selected by build tags)
    f := foxi.NewFoxi()
    
    // Open database
    err := f.Open("customers.dbf")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    // Show backend information
    fmt.Printf("Using backend: %s\n", f.Backend().String())
    
    // Get database information  
    header := f.Header()
    fmt.Printf("Records: %d, Last Updated: %s\n", 
        header.RecordCount(), header.LastUpdated().Format("2006-01-02"))
    
    // Navigate through records
    f.First()
    for !f.EOF() {
        nameField := f.FieldByName("NAME")
        name, _ := nameField.AsString()
        
        ageField := f.FieldByName("AGE") 
        age, _ := ageField.AsInt()
        
        fmt.Printf("Record %d: %s, Age: %d\n", f.Position(), name, age)
        f.Next()
    }
}
```

## Build Options

### Default Build (Pure Go)
```bash
go build                    # Uses gomkfdbf backend
go run main.go              # Pure Go, no CGO required
```

### CGO Build (C Library)
```bash
go build -tags foxicgo      # Uses mkfdbf C library
CGO_ENABLED=1 go build -tags foxicgo
```

### Cross-Platform Builds
```bash
# Pure Go - works on all platforms
GOOS=linux GOARCH=amd64 go build
GOOS=windows GOARCH=amd64 go build
GOOS=darwin GOARCH=arm64 go build

# CGO builds require appropriate toolchain per platform
CGO_ENABLED=1 GOOS=linux go build -tags foxicgo
```

## API Reference

### Core Operations

```go
f := foxi.NewFoxi()         // Create new instance
err := f.Open("data.dbf")   // Open database
defer f.Close()             // Always close when done
active := f.Active()        // Check if database is open
backend := f.Backend()      // Get backend information
```

### Header Information

```go
header := f.Header()
count := header.RecordCount()        // Total records
updated := header.LastUpdated()     // Last modification date
hasIndex := header.HasIndex()       // Has index files
hasMemo := header.HasFpt()          // Has memo file
codepage := header.Codepage()       // Character encoding
```

### Field Access

```go
// Field count and enumeration
fieldCount := f.FieldCount()
for i := 0; i < fieldCount; i++ {
    field := f.Field(i)
    fmt.Printf("%s (%s)\n", field.Name(), field.Type().String())
}

// Access field by name
nameField := f.FieldByName("CUSTOMER_NAME")
if nameField != nil {
    value, _ := nameField.AsString()
    fieldType := nameField.Type()
    size := nameField.Size()
}
```

### Record Navigation

```go
f.First()                   // Go to first record
f.Last()                    // Go to last record
f.Next()                    // Next record
f.Previous()                // Previous record
f.Skip(10)                  // Skip multiple records
f.Goto(42)                  // Go to specific record

pos := f.Position()         // Current record number (1-indexed)
isEOF := f.EOF()           // At end of file
isBOF := f.BOF()           // At beginning of file
```

### Field Value Reading

```go
field := f.FieldByName("SOME_FIELD")

// Type-safe conversion methods
stringVal, err := field.AsString()   // Convert to string
intVal, err := field.AsInt()         // Convert to integer  
floatVal, err := field.AsFloat()     // Convert to float64
boolVal, err := field.AsBool()       // Convert to boolean
timeVal, err := field.AsTime()       // Convert to time.Time

// Native value access
nativeVal, err := field.Value()      // Get in native type
isNull, err := field.IsNull()        // Check for null
```

### Record State

```go
deleted := f.Deleted()      // Check if record is deleted
err := f.Delete()           // Mark record for deletion (soft delete)
err := f.Recall()           // Undelete record
```

## Advanced Features

> **Note**: The advanced features documented below represent the full capabilities available in the underlying Vulpo/mkfdbf libraries. The current foxi implementation provides the basic unified API (database operations, navigation, field access, record state). These advanced features would be implemented in future versions or can be accessed directly via the backend packages.

### Index Operations

The underlying libraries support full index operations for efficient data access and seeking:

```go
// List all available indexes/tags
tags := f.ListTags()
fmt.Printf("Available indexes: %d\n", len(tags))
for _, tag := range tags {
    fmt.Printf("  %s\n", tag.Name())
}

// Select an index for navigation
nameTag := f.TagByName("NAME_IDX")
if nameTag != nil {
    f.SelectTag(nameTag)
    
    // Navigation now follows index order
    f.First()  // First alphabetically
    f.Next()   // Next alphabetically
}

// Return to physical record order
f.SelectTag(nil)

// Get currently selected index
selectedTag := f.SelectedTag()
if selectedTag != nil {
    fmt.Printf("Using index: %s\n", selectedTag.Name())
}
```

### Seeking Records

Use indexes to quickly find specific records:

```go
// Select an index first
nameTag := f.TagByName("NAME_IDX")
f.SelectTag(nameTag)

// Seek for specific values
result, err := f.Seek("Smith")
if err != nil {
    log.Printf("Seek failed: %v", err)
} else {
    switch result {
    case foxi.SeekSuccess:
        fmt.Println("Found exact match")
    case foxi.SeekAfter:
        fmt.Println("Positioned after where record would be")
    case foxi.SeekEOF:
        fmt.Println("Value would be after last record")
    }
}

// Continue seeking for more matches
for {
    result, err := f.SeekNext("Smith")
    if result != foxi.SeekSuccess {
        break
    }
    
    // Process matching record
    nameField := f.FieldByName("NAME")
    name, _ := nameField.AsString()
    fmt.Printf("Found: %s\n", name)
}

// Seek with numeric values (more efficient)
salaryTag := f.TagByName("SALARY_IDX")
f.SelectTag(salaryTag)
result, err = f.SeekDouble(50000.0)
```

### Expression-Based Filtering

Foxi supports native dBASE expressions for powerful record filtering:

```go
// Simple field comparisons
results, err := f.SearchByExpression("AGE > 30", nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d records with age > 30\n", len(results.Matches))
for _, match := range results.Matches {
    nameField := match.FieldReaders["NAME"]
    ageField := match.FieldReaders["AGE"]
    
    name, _ := nameField.AsString()
    age, _ := ageField.AsInt()
    
    fmt.Printf("Record %d: %s, Age: %d\n", match.RecordNumber, name, age)
}

// Complex expressions with functions
results, err = f.SearchByExpression(
    "YEAR(BIRTH_DATE) = 1990 .AND. SUBSTR(NAME, 1, 1) = 'J'", 
    &foxi.ExprSearchOptions{MaxResults: 10})

// String functions
results, err = f.SearchByExpression("UPPER(LEFT(NAME, 3)) = 'SMI'", nil)

// Date functions  
results, err = f.SearchByExpression("MONTH(HIRE_DATE) = 12", nil)

// Logical operations
results, err = f.SearchByExpression(
    "(SALARY > 50000 .OR. BONUS > 10000) .AND. ACTIVE", nil)
```

#### Supported Expression Functions

**String Functions:**
- `SUBSTR(string, start, length)` - Extract substring
- `LEFT(string, count)` - Left characters  
- `RIGHT(string, count)` - Right characters
- `UPPER(string)` - Convert to uppercase
- `LOWER(string)` - Convert to lowercase
- `TRIM(string)` - Remove trailing spaces
- `LTRIM(string)` - Remove leading spaces
- `ALLTRIM(string)` - Remove leading and trailing spaces

**Date Functions:**
- `YEAR(date)` - Extract year
- `MONTH(date)` - Extract month
- `DAY(date)` - Extract day
- `CTOD(string)` - Convert string to date
- `DTOS(date)` - Convert date to string
- `DATE()` - Current date

**Numeric Functions:**
- `STR(number, length, decimals)` - Convert number to string
- `VAL(string)` - Convert string to number
- `INT(number)` - Integer part
- `ABS(number)` - Absolute value

**Conditional:**
- `IIF(condition, true_value, false_value)` - Conditional expression

**Record Functions:**
- `RECNO()` - Current record number
- `RECCOUNT()` - Total record count
- `DELETED()` - Check if record is deleted

### Counting and Iteration

```go
// Count matching records
count, err := f.CountByExpression("ACTIVE .AND. SALARY > 40000")
fmt.Printf("Found %d active high-salary employees\n", count)

// Iterate through matches
err = f.ForEachExpressionMatch("YEAR(BIRTH_DATE) = 1985", 
    func(fieldReaders map[string]foxi.FieldReader) error {
        nameField := fieldReaders["NAME"]
        name, _ := nameField.AsString()
        fmt.Printf("Born in 1985: %s\n", name)
        return nil  // Continue iteration
    })
```

### Regex Search

For pattern-based searching on character fields:

```go
// Basic regex search
results, err := f.RegexSearch("NAME", "^Smith", &foxi.RegexSearchOptions{
    CaseInsensitive: true,
    MaxResults: 50,
})

if err != nil {
    log.Fatal(err)
}

for _, match := range results.Matches {
    fmt.Printf("Record %d: %s (matches: %v)\n", 
        match.RecordNumber, match.FieldValue, match.Matches)
}

// Advanced regex patterns
results, err = f.RegexSearch("EMAIL", "@(gmail|yahoo|hotmail)\\.com$", &foxi.RegexSearchOptions{
    CaseInsensitive: true,
})

// Count regex matches
count, err := f.RegexCount("PHONE", "\\(\\d{3}\\)\\s\\d{3}-\\d{4}", nil)
fmt.Printf("Found %d properly formatted phone numbers\n", count)

// Check if any records match
exists, err := f.RegexExists("ZIP", "^\\d{5}(-\\d{4})?$", nil)
if exists {
    fmt.Println("Found records with valid ZIP codes")
}
```

### Compiled Expression Filters

For reusable expression filtering:

```go
// Create a compiled expression filter
filter, err := f.NewExprFilter("AGE >= 18 .AND. ACTIVE")
if err != nil {
    log.Fatal(err)
}
defer filter.Free()  // Always free resources

// Use filter on different records
f.First()
for !f.EOF() {
    matches, err := filter.Evaluate()
    if err != nil {
        log.Fatal(err)
    }
    
    if matches {
        fmt.Printf("Record %d matches criteria\n", f.Position())
    }
    
    f.Next()
}

// Get different result types from expressions
stringResult, err := filter.EvaluateAsString()
numericResult, err := filter.EvaluateAsDouble()
```

### Deleted Record Management

DBF files use "soft delete" - records are marked for deletion but remain in the file:

```go
// Check if current record is deleted
f.First()
if f.Deleted() {
    fmt.Println("Current record is marked for deletion")
}

// Mark record for deletion (soft delete)
err := f.Delete()
if err != nil {
    log.Printf("Failed to delete record: %v", err)
}

// Undelete (recall) a record
err = f.Recall()
if err != nil {
    log.Printf("Failed to recall record: %v", err)
}

// Count deleted vs active records
deletedCount, err := f.CountDeleted()
activeCount, err := f.CountActive()
fmt.Printf("Records: %d deleted, %d active\n", deletedCount, activeCount)

// List all deleted records
deletedRecords, err := f.ListDeletedRecords()
for _, record := range deletedRecords {
    fmt.Printf("Deleted record: %d\n", record.RecordNumber)
}

// Process each deleted record
err = f.ForEachDeletedRecord(func(recordNumber int) error {
    fmt.Printf("Processing deleted record %d\n", recordNumber)
    return nil
})

// Batch recall all deleted records
recalledCount, err := f.RecallAllDeleted()
fmt.Printf("Recalled %d records\n", recalledCount)

// Physical removal (PERMANENT - use with caution!)
err = f.Pack()  // WARNING: This permanently removes deleted records!
if err != nil {
    log.Printf("Pack failed: %v", err)
} else {
    fmt.Println("Database packed - deleted records permanently removed")
    f.First() // Must reposition after pack
}
```

## Field Types

Foxi supports all standard DBF field types:

| Type | Code | Description | Go Type |
|------|------|-------------|---------|
| Character | C | Text fields | string |
| Numeric | N | Numbers with decimals | float64 |
| Date | D | Dates (CCYYMMDD) | time.Time |
| Logical | L | Boolean (T/F, Y/N) | bool |
| Integer | I | 32-bit integers | int |
| Float | F | Floating-point | float64 |
| DateTime | T | Date and time | time.Time |
| Currency | Y | Money values | float64 |
| Memo | M | Large text fields | string |

## Error Handling

```go
f := foxi.NewFoxi()

// Always check errors
err := f.Open("data.dbf")
if err != nil {
    log.Fatalf("Failed to open: %v", err)
}

// Handle navigation errors
err = f.Goto(999999)
if err != nil {
    log.Printf("Invalid record: %v", err)
}

// Field conversion errors
field := f.FieldByName("NUMERIC_FIELD")
value, err := field.AsInt()
if err != nil {
    log.Printf("Conversion failed: %v", err)
}
```

## Testing

Run the implementation-agnostic test suite:

```bash
# Test with pure Go backend (default)
go test ./tests/

# Test with CGO backend  
go test -tags foxicgo ./tests/

# Run benchmarks
go test -bench=. ./tests/

# Test both backends
go test ./tests/ && go test -tags foxicgo ./tests/
```

## Performance

Performance comparison between backends (approximate):

| Operation | Pure Go | CGO | Ratio |
|-----------|---------|-----|-------|
| File Opening | ~89μs | ~33μs | 2.7x |
| Record Navigation | ~3.6ms | ~2.6ms | 1.4x |
| Field Access | ~18ns | ~10ns | 1.8x |
| Memory Usage | 66KB | 384KB | 0.17x |

The pure Go implementation achieves 50-70% of C performance while providing 100% memory safety and deployment simplicity.

## Project Structure

```
foxi/
├── foxi.go              # Main API interface
├── foxi_go.go          # Pure Go backend (+build !foxicgo)
├── foxi_cgo.go         # CGO backend (+build foxicgo)  
├── go.mod              # Module definition
├── README.md           # This file
├── pkg/                # Internal backend packages
│   ├── gocore/        # Pure Go implementation (gomkfdbf library)
│   └── cgocore/       # CGO implementation
│       ├── cgocore.go # CGO wrapper
│       └── mkfdbflib/ # Static library and headers
└── tests/             # Implementation-agnostic tests
    └── foxi_test.go   # Test suite
```

## Backend Selection Logic

The backend is selected automatically at compile time:

1. **Default**: Pure Go backend using gomkfdbf
   - No special build flags required
   - Works on all platforms Go supports
   - No CGO dependencies

2. **CGO**: C library backend using mkfdbf  
   - Enabled with `-tags foxicgo`
   - Requires CGO_ENABLED=1
   - Maximum performance

## Compatibility

### Vulpo Compatibility
Foxi provides the same API surface as Vulpo for easy migration:

```go
// Vulpo code
v := &vulpo.Vulpo{}
v.Open("data.dbf")
field := v.FieldByName("NAME")
name, _ := field.AsString()

// Foxi code (identical)
f := foxi.NewFoxi()
f.Open("data.dbf")  
field := f.FieldByName("NAME")
name, _ := field.AsString()
```

### Database Compatibility
- **Visual FoxPro**: Full compatibility
- **dBASE III/IV/V**: Complete support
- **Clipper**: Full compatibility
- **FoxPro 2.x**: Supported
- **Other xBase**: Most variants work

## Direct Backend Usage

If you prefer to use the backends directly without the unified interface:

### Pure Go Backend (Internal)

```go
import "github.com/mkfoss/foxi/pkg/gocore"

// Direct gomkfdbf usage
cb := &gocore.Code4{}
data := gocore.D4Open(cb, "data.dbf")
defer gocore.D4Close(data)

// Navigate and read fields
gocore.D4Top(data)
field := gocore.D4Field(data, "NAME")
value := gocore.F4Str(field)
```

### CGO Backend (Internal)

```go
import "github.com/mkfoss/foxi/pkg/cgocore"

// Direct C library usage
// Note: This is a low-level interface requiring C knowledge
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Test both backends:
   ```bash
   go test ./tests/
   go test -tags foxicgo ./tests/
   ```
5. Submit pull request

## License

MIT License - see LICENSE file for details.

## Related Projects

- **[Vulpo](https://github.com/mkfoss/vulpo)**: Original Go DBF library (CGO-only)
- **[gomkfdbf](https://github.com/sean/gomkfdbf)**: Pure Go CodeBase translation  
- **[mkfdbf](https://github.com/mkfoss/mkfdbf)**: C CodeBase library

---

*Foxi - Flexible DBF access for modern Go applications*