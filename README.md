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

### Index Operations

Foxi provides comprehensive index support with lazy loading for efficient database access:

```go
// Access the indexes collection (lazy loaded)
indexes := f.Indexes()

// List all available indexes
indexList := indexes.List()
fmt.Printf("Available indexes: %d\n", len(indexList))
for _, index := range indexList {
    fmt.Printf("  Index: %s, Tags: %d\n", index.Name(), index.TagCount())
}

// List all available tags from all indexes
tags := indexes.Tags()
fmt.Printf("Available tags: %d\n", len(tags))
for _, tag := range tags {
    fmt.Printf("  Tag: %s, Expression: %s, KeyLen: %d\n", 
        tag.Name(), tag.Expression(), tag.KeyLength())
}

// Select a tag for navigation
nameTag := indexes.TagByName("NAME_IDX")
if nameTag != nil {
    err := indexes.SelectTag(nameTag)
    if err == nil {
        // Navigation now follows index order
        f.First()  // First in index order
        f.Next()   // Next in index order
    }
}

// Return to physical record order
indexes.SelectTag(nil)

// Get currently selected tag
selectedTag := indexes.SelectedTag()
if selectedTag != nil {
    fmt.Printf("Using tag: %s\n", selectedTag.Name())
}
```

### Seeking Records

Use tags to quickly find specific records:

```go
// Get a tag for seeking
indexes := f.Indexes()
nameTag := indexes.TagByName("NAME_IDX")
if nameTag == nil {
    log.Fatal("NAME_IDX tag not found")
}

// Seek for specific values
result, err := nameTag.SeekString("Smith")
if err != nil {
    log.Printf("Seek failed: %v", err)
} else {
    switch result {
    case foxi.SeekSuccess:
        fmt.Println("Found exact match")
        // Record is now positioned at the matching entry
        nameField := f.FieldByName("NAME")
        name, _ := nameField.AsString()
        fmt.Printf("Found: %s at record %d\n", name, f.Position())
    case foxi.SeekAfter:
        fmt.Println("Positioned after where record would be")
    case foxi.SeekEOF:
        fmt.Println("Value would be after last record")
    }
}

// Seek with different data types
result, err = nameTag.Seek("Smith")        // Generic seek
result, err = nameTag.SeekInt(12345)       // Integer seek
result, err = nameTag.SeekDouble(50000.0)  // Float64 seek

// Tag-specific navigation
if result == foxi.SeekSuccess {
    err = nameTag.Next()     // Next in tag order
    err = nameTag.Previous() // Previous in tag order
    
    key := nameTag.CurrentKey()     // Current index key value
    recNo := nameTag.RecordNumber() // Current record number
    pos := nameTag.Position()       // Position as percentage (0.0-1.0)
}
```

### Current Implementation Status

The current foxi implementation provides:

✅ **Implemented Features:**
- Lazy-loaded index discovery and access
- Index and tag enumeration
- Tag selection for record ordering
- Basic seek operations (SeekString, SeekInt, SeekDouble, generic Seek)
- Tag-based navigation (First, Last, Next, Previous)
- Position-based access (Position, PositionSet)
- Tag properties (Name, Expression, KeyLength, IsUnique, IsDescending)
- Current record information (RecordNumber, CurrentKey, EOF, BOF)

🚧 **Future Enhancements:**
- Full expression evaluation for CurrentKey()
- Advanced seek operations (SeekNext for duplicates)
- Expression-based filtering
- Regex search capabilities
- Compiled expression filters

The basic index functionality is fully operational and provides substantial performance benefits for record navigation and seeking.

## Deleted Record Management

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

// Additional deleted record operations would be available
// in future versions of foxi
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