// Package foxi provides a Go interface to DBF (dBASE) files with support for both
// CGO-based (mkfdbf) and pure Go (gomkfdbf) backends via build tags.
//
// Build Tags:
//   - Default: Uses pure Go implementation (gomkfdbf)
//   - -tags foxicgo: Uses CGO implementation with mkfdbf C library
//
// The package offers comprehensive functionality for reading, writing, navigating, 
// and manipulating DBF databases with support for indexes, field types, deleted records,
// and expression filtering.
//
// Basic usage:
//
//	f := &foxi.Foxi{}
//	err := f.Open("data.dbf")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer f.Close()
//
//	// Navigate and read records
//	f.First(0)
//	for !f.EOF() {
//		nameField := f.FieldByName("NAME")
//		name, _ := nameField.AsString()
//		fmt.Printf("%s\n", name)
//		f.Next()
//	}
package foxi

import (
	"fmt"
	"time"
)

// Backend represents the implementation backend type
type Backend int

const (
	BackendPureGo Backend = iota // Default: Pure Go implementation (gomkfdbf)
	BackendCGO                   // CGO implementation (mkfdbf C library)
)

// String returns the backend name
func (b Backend) String() string {
	switch b {
	case BackendPureGo:
		return "Pure Go (gomkfdbf)"
	case BackendCGO:
		return "CGO (mkfdbf)"
	default:
		return "Unknown"
	}
}

// Foxi represents a connection to a DBF database file with automatic backend selection.
// The implementation is chosen at compile time via build tags:
//   - Default build: Uses pure Go backend (gomkfdbf)
//   - Build with -tags foxicgo: Uses CGO backend (mkfdbf)
//
// All operations require an active connection (Open() called successfully).
// The connection is automatically closed when the Foxi instance is garbage collected,
// but Close() should be called explicitly for proper resource management.
//
// Create instances using NewFoxi() function.
type Foxi struct {
	impl foxiImpl // Backend implementation (selected by build tags)
}

// NewFoxi creates a new Foxi instance with the appropriate backend.
// The backend is selected at compile time via build tags.
// This function is implemented in the backend-specific files (foxi_go.go or foxi_cgo.go).

// foxiImpl defines the internal interface that both backends must implement
type foxiImpl interface {
	// Database operations
	Open(filename string) error
	Close() error
	Active() bool

	// Header information
	Header() Header

	// Field access
	Fields() *Fields
	FieldCount() int
	Field(index int) Field
	FieldByName(name string) Field

	// Navigation
	Goto(recordNumber int) error
	First() error
	Last() error
	Next() error
	Previous() error
	Skip(count int) error
	Position() int
	EOF() bool
	BOF() bool

	// Record state
	Deleted() bool
	Delete() error
	Recall() error

	// Index operations
	Indexes() *Indexes

	// Backend information
	Backend() Backend
}

// Open establishes a connection to the specified DBF file.
// The filename should include the full path and .dbf extension.
func (f *Foxi) Open(filename string) error {
	return f.impl.Open(filename)
}

// Close closes the database connection and releases all associated resources.
// After Close() is called, the Foxi instance can be reused by calling Open()
// with a new filename.
func (f *Foxi) Close() error {
	return f.impl.Close()
}

// Active reports whether the database connection is active and ready for use.
func (f *Foxi) Active() bool {
	return f.impl.Active()
}

// Header returns the database file header information.
func (f *Foxi) Header() Header {
	return f.impl.Header()
}

// Fields returns the field collection for all fields in the database.
func (f *Foxi) Fields() *Fields {
	return f.impl.Fields()
}

// FieldCount returns the total number of fields in the database.
func (f *Foxi) FieldCount() int {
	return f.impl.FieldCount()
}

// Field returns the field instance at the specified zero-based index.
func (f *Foxi) Field(index int) Field {
	return f.impl.Field(index)
}

// FieldByName returns the field instance with the specified name.
// The lookup is case-insensitive.
func (f *Foxi) FieldByName(name string) Field {
	return f.impl.FieldByName(name)
}

// Goto moves to the specified record number (1-indexed).
func (f *Foxi) Goto(recordNumber int) error {
	return f.impl.Goto(recordNumber)
}

// First moves to the first record in the current order.
func (f *Foxi) First() error {
	return f.impl.First()
}

// Last moves to the last record in the current order.
func (f *Foxi) Last() error {
	return f.impl.Last()
}

// Next moves to the next record.
func (f *Foxi) Next() error {
	return f.impl.Next()
}

// Previous moves to the previous record.
func (f *Foxi) Previous() error {
	return f.impl.Previous()
}

// Skip skips the specified number of records (positive = forward, negative = backward).
func (f *Foxi) Skip(count int) error {
	return f.impl.Skip(count)
}

// Position returns the current record number (1-indexed).
func (f *Foxi) Position() int {
	return f.impl.Position()
}

// EOF returns true if positioned at end of file.
func (f *Foxi) EOF() bool {
	return f.impl.EOF()
}

// BOF returns true if positioned at beginning of file.
func (f *Foxi) BOF() bool {
	return f.impl.BOF()
}

// Deleted returns true if the current record is marked for deletion.
func (f *Foxi) Deleted() bool {
	return f.impl.Deleted()
}

// Delete marks the current record for deletion (soft delete).
func (f *Foxi) Delete() error {
	return f.impl.Delete()
}

// Recall undeletes the current record.
func (f *Foxi) Recall() error {
	return f.impl.Recall()
}

// ==========================================================================
// MUST VARIANTS - Panic instead of returning errors
// ==========================================================================

// MustOpen establishes a connection to the specified DBF file.
// Panics if the operation fails.
func (f *Foxi) MustOpen(filename string) {
	if err := f.Open(filename); err != nil {
		panic(err)
	}
}

// MustGoto moves to the specified record number (1-indexed).
// Panics if the operation fails.
func (f *Foxi) MustGoto(recordNumber int) {
	if err := f.Goto(recordNumber); err != nil {
		panic(err)
	}
}

// MustFirst moves to the first record in the current order.
// Panics if the operation fails.
func (f *Foxi) MustFirst() {
	if err := f.First(); err != nil {
		panic(err)
	}
}

// MustLast moves to the last record in the current order.
// Panics if the operation fails.
func (f *Foxi) MustLast() {
	if err := f.Last(); err != nil {
		panic(err)
	}
}

// MustNext moves to the next record.
// Panics if the operation fails.
func (f *Foxi) MustNext() {
	if err := f.Next(); err != nil {
		panic(err)
	}
}

// MustPrevious moves to the previous record.
// Panics if the operation fails.
func (f *Foxi) MustPrevious() {
	if err := f.Previous(); err != nil {
		panic(err)
	}
}

// MustSkip skips the specified number of records (positive = forward, negative = backward).
// Panics if the operation fails.
func (f *Foxi) MustSkip(count int) {
	if err := f.Skip(count); err != nil {
		panic(err)
	}
}

// MustDelete marks the current record for deletion (soft delete).
// Panics if the operation fails.
func (f *Foxi) MustDelete() {
	if err := f.Delete(); err != nil {
		panic(err)
	}
}

// MustRecall undeletes the current record.
// Panics if the operation fails.
func (f *Foxi) MustRecall() {
	if err := f.Recall(); err != nil {
		panic(err)
	}
}

// Indexes returns the index collection with lazy loading support.
// Indexes are not loaded until first access.
func (f *Foxi) Indexes() *Indexes {
	return f.impl.Indexes()
}

// Backend returns information about which backend implementation is being used.
func (f *Foxi) Backend() Backend {
	return f.impl.Backend()
}

// Header contains metadata about the DBF file
type Header struct {
	recordCount uint
	lastUpdated time.Time
	hasIndex    bool
	hasFpt      bool
	codepage    Codepage
}

// RecordCount returns the total number of records in the database.
func (h *Header) RecordCount() uint {
	return h.recordCount
}

// LastUpdated returns the date when the database was last modified.
func (h *Header) LastUpdated() time.Time {
	return h.lastUpdated
}

// HasIndex returns true if the database has associated index files.
func (h *Header) HasIndex() bool {
	return h.hasIndex
}

// HasFpt returns true if the database has a memo file (.FPT).
func (h *Header) HasFpt() bool {
	return h.hasFpt
}

// Codepage returns the character encoding used by the database.
func (h *Header) Codepage() Codepage {
	return h.codepage
}

// Field defines the interface for accessing both field definition information
// and field value reading capabilities from the current record.
type Field interface {
	// Value returns the field's native value in its appropriate Go type
	Value() (interface{}, error)

	// Type conversion methods
	AsString() (string, error)
	AsInt() (int, error)
	AsFloat() (float64, error)
	AsBool() (bool, error)
	AsTime() (time.Time, error)

	// Null checking
	IsNull() (bool, error)

	// Must variants - panic instead of returning errors
	MustValue() interface{}
	MustAsString() string
	MustAsInt() int
	MustAsFloat() float64
	MustAsBool() bool
	MustAsTime() time.Time
	MustIsNull() bool

	// Field definition methods
	Name() string
	Type() FieldType
	Size() uint8
	Decimals() uint8
	IsSystem() bool
	IsNullable() bool
	IsBinary() bool
}

// Fields provides access to the database field collection
type Fields struct {
	fields  []Field
	indices map[string]int // name -> index mapping (case-insensitive)
}

// Count returns the total number of fields in the database.
func (f *Fields) Count() int {
	return len(f.fields)
}

// ByIndex returns the field at the specified zero-based index.
func (f *Fields) ByIndex(index int) Field {
	if index < 0 || index >= len(f.fields) {
		return nil
	}
	return f.fields[index]
}

// ByName returns the field with the specified name (case-insensitive).
func (f *Fields) ByName(name string) Field {
	if f.indices == nil {
		return nil
	}
	
	index, exists := f.indices[name]
	if !exists {
		return nil
	}
	
	return f.ByIndex(index)
}

// FieldType represents the data type of a database field
type FieldType int

const (
	FTUnknown   FieldType = iota
	FTCharacter           // C - Character/String
	FTNumeric             // N - Numeric
	FTLogical             // L - Logical/Boolean
	FTDate                // D - Date
	FTInteger             // I - Integer (32-bit)
	FTDateTime            // T - DateTime
	FTCurrency            // Y - Currency
	FTMemo                // M - Memo
	FTBlob                // B - Binary/Blob (deprecated)
	FTFloat               // F - Float
	FTGeneral             // G - General (OLE object)
	FTPicture             // P - Picture (OLE object)
	FTVarBinary           // Q - VarBinary
	FTVarchar             // V - Varchar
	FTTimestamp           // W - Timestamp (not standard)
	FTDouble              // X - Double (not standard)
)

// String returns the single-character field type identifier
func (ft FieldType) String() string {
	fieldTypes := "CNLDITYMBFGPQVWX"
	if ft >= 1 && int(ft) <= len(fieldTypes) {
		return string(fieldTypes[ft-1])
	}
	return "unknown"
}

// Name returns the descriptive name of the field type
func (ft FieldType) Name() string {
	switch ft {
	case FTCharacter:
		return "character"
	case FTNumeric:
		return "numeric"
	case FTLogical:
		return "logical"
	case FTDate:
		return "date"
	case FTInteger:
		return "integer"
	case FTDateTime:
		return "datetime"
	case FTCurrency:
		return "currency"
	case FTMemo:
		return "memo"
	case FTBlob:
		return "blob"
	case FTFloat:
		return "float"
	case FTGeneral:
		return "general"
	case FTPicture:
		return "picture"
	case FTVarBinary:
		return "varbinary"
	case FTVarchar:
		return "varchar"
	case FTTimestamp:
		return "timestamp"
	case FTDouble:
		return "double"
	default:
		return "unknown"
	}
}

// Codepage represents character encoding information
type Codepage uint8

// String returns the codepage description
func (c Codepage) String() string {
	switch c {
	case 0x03:
		return "Windows ANSI (1252)"
	case 0x01:
		return "U.S. MS-DOS (437)"
	case 0x02:
		return "International MS-DOS (850)"
	default:
		return "Unknown Codepage"
	}
}

// =========================================================================
// INDEX SUPPORT
// =========================================================================

// Indexes provides lazy-loaded access to database indexes
type Indexes struct {
	impl indexesImpl // Backend implementation
}

// indexesImpl defines the interface for backend-specific index operations
type indexesImpl interface {
	// Core operations
	Load() error
	Count() int
	Loaded() bool

	// Access methods
	ByIndex(index int) Index
	ByName(name string) Index
	List() []Index

	// Tag operations
	TagByName(name string) Tag
	SelectedTag() Tag
	SelectTag(tag Tag) error
	Tags() []Tag
}

// Load loads all available indexes from the database files.
// This is called automatically on first access but can be called explicitly.
func (idx *Indexes) Load() error {
	if idx.impl == nil {
		return fmt.Errorf("indexes not initialized")
	}
	return idx.impl.Load()
}

// Count returns the number of loaded indexes.
func (idx *Indexes) Count() int {
	if idx.impl == nil {
		return 0
	}
	return idx.impl.Count()
}

// Loaded returns true if indexes have been loaded from disk.
func (idx *Indexes) Loaded() bool {
	if idx.impl == nil {
		return false
	}
	return idx.impl.Loaded()
}

// ByIndex returns the index at the specified position.
func (idx *Indexes) ByIndex(index int) Index {
	if idx.impl == nil {
		return nil
	}
	// Auto-load on first access
	if !idx.impl.Loaded() {
		if err := idx.impl.Load(); err != nil {
			return nil
		}
	}
	return idx.impl.ByIndex(index)
}

// ByName returns the index with the specified name.
func (idx *Indexes) ByName(name string) Index {
	if idx.impl == nil {
		return nil
	}
	// Auto-load on first access
	if !idx.impl.Loaded() {
		if err := idx.impl.Load(); err != nil {
			return nil
		}
	}
	return idx.impl.ByName(name)
}

// List returns all available indexes.
func (idx *Indexes) List() []Index {
	if idx.impl == nil {
		return nil
	}
	// Auto-load on first access
	if !idx.impl.Loaded() {
		if err := idx.impl.Load(); err != nil {
			return nil
		}
	}
	return idx.impl.List()
}

// TagByName returns the tag with the specified name.
func (idx *Indexes) TagByName(name string) Tag {
	if idx.impl == nil {
		return nil
	}
	// Auto-load on first access
	if !idx.impl.Loaded() {
		if err := idx.impl.Load(); err != nil {
			return nil
		}
	}
	return idx.impl.TagByName(name)
}

// SelectedTag returns the currently selected tag for record ordering.
func (idx *Indexes) SelectedTag() Tag {
	if idx.impl == nil {
		return nil
	}
	return idx.impl.SelectedTag()
}

// SelectTag sets the active tag for record navigation order.
// Pass nil to use natural record order (no index).
func (idx *Indexes) SelectTag(tag Tag) error {
	if idx.impl == nil {
		return fmt.Errorf("indexes not initialized")
	}
	return idx.impl.SelectTag(tag)
}

// Tags returns all available tags from all indexes.
func (idx *Indexes) Tags() []Tag {
	if idx.impl == nil {
		return nil
	}
	// Auto-load on first access
	if !idx.impl.Loaded() {
		if err := idx.impl.Load(); err != nil {
			return nil
		}
	}
	return idx.impl.Tags()
}

// MustLoad loads all available indexes from the database files.
// Panics if the operation fails.
func (idx *Indexes) MustLoad() {
	if err := idx.Load(); err != nil {
		panic(err)
	}
}

// MustSelectTag sets the active tag for record navigation order.
// Pass nil to use natural record order (no index).
// Panics if the operation fails.
func (idx *Indexes) MustSelectTag(tag Tag) {
	if err := idx.SelectTag(tag); err != nil {
		panic(err)
	}
}

// Index represents a database index file (e.g., .CDX file)
type Index interface {
	// Properties
	Name() string
	FileName() string
	TagCount() int
	
	// Access
	Tag(index int) Tag
	TagByName(name string) Tag
	Tags() []Tag
	
	// State
	IsOpen() bool
	IsProduction() bool
}

// Tag represents an index tag within an index file
type Tag interface {
	// Properties
	Name() string
	Expression() string
	Filter() string
	KeyLength() int
	
	// Attributes
	IsUnique() bool
	IsDescending() bool
	IsSelected() bool
	
	// Operations
	Seek(value interface{}) (SeekResult, error)
	SeekString(value string) (SeekResult, error)
	SeekDouble(value float64) (SeekResult, error)
	SeekInt(value int) (SeekResult, error)
	
	// Navigation (when tag is selected)
	First() error
	Last() error
	Next() error
	Previous() error
	Position() float64
	PositionSet(percent float64) error
	
	// Must variants for operations - panic instead of returning errors
	MustSeek(value interface{}) SeekResult
	MustSeekString(value string) SeekResult
	MustSeekDouble(value float64) SeekResult
	MustSeekInt(value int) SeekResult
	MustFirst()
	MustLast()
	MustNext()
	MustPrevious()
	MustPositionSet(percent float64)
	
	// Current state
	CurrentKey() string
	RecordNumber() int
	EOF() bool
	BOF() bool
}

// SeekResult indicates the result of a seek operation
type SeekResult int

const (
	SeekSuccess SeekResult = iota // Exact match found
	SeekAfter                     // Positioned after where record would be
	SeekEOF                       // Would be after last record
)

// String returns the seek result description
func (sr SeekResult) String() string {
	switch sr {
	case SeekSuccess:
		return "success"
	case SeekAfter:
		return "after"
	case SeekEOF:
		return "eof"
	default:
		return "unknown"
	}
}
