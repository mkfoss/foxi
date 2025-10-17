//go:build foxicgo
// +build foxicgo

package foxi

/*
#cgo CFLAGS: -I./pkg/cgocore/mkfdbflib
#cgo LDFLAGS: -L./pkg/cgocore/mkfdbflib -lmkfdbf
#include "d4all.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

// cgoImpl implements foxiImpl using the mkfdbf C library via CGO
type cgoImpl struct {
	codeBase *C.CODE4
	data     *C.DATA4
	fields   *Fields
	indexes  *Indexes
	filename string
}

// NewFoxi creates a new Foxi instance with CGO backend
func NewFoxi() *Foxi {
	impl := &cgoImpl{}
	return &Foxi{impl: impl}
}

// Open establishes a connection to the specified DBF file using mkfdbf C library
func (c *cgoImpl) Open(filename string) error {
	if c.data != nil {
		return fmt.Errorf("database already open")
	}

	// Initialize CODE4 structure
	c.codeBase = (*C.CODE4)(C.malloc(C.sizeof_CODE4))
	if c.codeBase == nil {
		return fmt.Errorf("failed to allocate CODE4 structure")
	}

	// Initialize the codebase using code4initLow (code4init macro expansion)
	result := C.code4initLow(c.codeBase, nil, 6401, C.long(C.sizeof_CODE4))
	if result != 0 {
		C.free(unsafe.Pointer(c.codeBase))
		c.codeBase = nil
		return fmt.Errorf("failed to initialize codebase: %d", int(result))
	}

	// Convert Go string to C string
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	// Open the data file
	c.data = C.d4open(c.codeBase, cFilename)
	if c.data == nil {
		// Clean up on failure
		C.code4initUndo(c.codeBase)
		C.free(unsafe.Pointer(c.codeBase))
		c.codeBase = nil
		return fmt.Errorf("failed to open database file: %s", filename)
	}

	c.filename = filename

	// Set finalizer to ensure cleanup
	runtime.SetFinalizer(c, (*cgoImpl).finalize)

	// Build Fields collection from C data
	err := c.buildFields()
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

// Close closes the database connection and releases resources
func (c *cgoImpl) Close() error {
	if c.data == nil {
		return nil
	}

	// Remove finalizer since we've cleaned up manually
	runtime.SetFinalizer(c, nil)

	return c.reset()
}

// finalize is called by the garbage collector to ensure cleanup
func (c *cgoImpl) finalize() {
	if c.data != nil {
		// Best effort cleanup - ignore errors since we can't return them
		_ = c.Close()
	}
}

func (c *cgoImpl) reset() error {
	// Close the data file
	if c.data != nil {
		result := C.d4close(c.data)
		c.data = nil
		if result != 0 {
			return fmt.Errorf("failed to close database: %d", int(result))
		}
	}

	// Cleanup the codebase
	if c.codeBase != nil {
		C.code4initUndo(c.codeBase)
		C.free(unsafe.Pointer(c.codeBase))
		c.codeBase = nil
	}

	// Clear all state
	c.filename = ""
	c.fields = nil

	return nil
}

// Active reports whether the database connection is active
func (c *cgoImpl) Active() bool {
	return c.data != nil
}

// Header returns database header information
func (c *cgoImpl) Header() Header {
	if c.data == nil {
		return Header{}
	}

	// Read header information from C library
	recordCount := uint(C.d4recCountDo(c.data))

	header := Header{
		recordCount: recordCount,
		hasIndex:    false,          // Will be detected based on actual indexes
		hasFpt:      false,          // Will be detected based on memo fields
		codepage:    Codepage(0x03), // Default to Windows ANSI
	}

	// Read the DBF file header directly to get date information
	dataFile := c.data.dataFile
	if dataFile != nil {
		// Read the first 32 bytes of the DBF file header directly
		headerBytes := make([]byte, 32)
		result := C.file4read(&dataFile.file, 0, unsafe.Pointer(&headerBytes[0]), 32)
		if result == 32 {
			// Parse date from file header (bytes 1-3)
			year := int(headerBytes[1])
			if year < 80 {
				year += 2000 // Y2K handling
			} else {
				year += 1900
			}
			month := int(headerBytes[2])
			day := int(headerBytes[3])

			if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
				header.lastUpdated = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			}

			// Read codepage from header (byte 29)
			if len(headerBytes) > 29 {
				header.codepage = Codepage(headerBytes[29])
			}
		}
	}

	return header
}

// Fields returns the field collection
func (c *cgoImpl) Fields() *Fields {
	return c.fields
}

// FieldCount returns the number of fields
func (c *cgoImpl) FieldCount() int {
	if c.fields == nil {
		return 0
	}
	return c.fields.Count()
}

// Field returns field by index
func (c *cgoImpl) Field(index int) Field {
	if c.fields == nil {
		return nil
	}
	return c.fields.ByIndex(index)
}

// FieldByName returns field by name
func (c *cgoImpl) FieldByName(name string) Field {
	if c.fields == nil {
		return nil
	}
	return c.fields.ByName(name)
}

// Navigation methods
func (c *cgoImpl) Goto(recordNumber int) error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	result := C.d4go(c.data, C.long(recordNumber))
	if result != 0 {
		return fmt.Errorf("failed to goto record %d", recordNumber)
	}
	return nil
}

func (c *cgoImpl) First() error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	result := C.d4top(c.data)
	if result != 0 {
		return fmt.Errorf("failed to go to first record")
	}
	return nil
}

func (c *cgoImpl) Last() error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	result := C.d4bottom(c.data)
	if result != 0 {
		return fmt.Errorf("failed to go to last record")
	}
	return nil
}

func (c *cgoImpl) Next() error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	result := C.d4skip(c.data, 1)
	if result != 0 {
		return fmt.Errorf("failed to move to next record")
	}
	return nil
}

func (c *cgoImpl) Previous() error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	result := C.d4skip(c.data, -1)
	if result != 0 {
		return fmt.Errorf("failed to move to previous record")
	}
	return nil
}

func (c *cgoImpl) Skip(count int) error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	result := C.d4skip(c.data, C.long(count))
	if result != 0 {
		return fmt.Errorf("failed to skip %d records", count)
	}
	return nil
}

func (c *cgoImpl) Position() int {
	if c.data == nil {
		return 0
	}

	return int(C.d4recNo(c.data))
}

func (c *cgoImpl) EOF() bool {
	if c.data == nil {
		return true
	}

	return C.d4eof(c.data) != 0
}

func (c *cgoImpl) BOF() bool {
	if c.data == nil {
		return true
	}

	return C.d4bof(c.data) != 0
}

// Record state methods
func (c *cgoImpl) Deleted() bool {
	if c.data == nil {
		return false
	}

	return C.d4deleted(c.data) != 0
}

func (c *cgoImpl) Delete() error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	C.d4delete(c.data)
	return nil
}

func (c *cgoImpl) Recall() error {
	if c.data == nil {
		return fmt.Errorf("database not open")
	}

	C.d4recall(c.data)
	return nil
}

// Indexes returns the index collection
func (c *cgoImpl) Indexes() *Indexes {
	if c.indexes == nil {
		c.indexes = &Indexes{
			impl: &cgoIndexesImpl{
				data:   c.data,
				loaded: false,
			},
		}
	}
	return c.indexes
}

// Backend returns the backend type
func (c *cgoImpl) Backend() Backend {
	return BackendCGO
}

// buildFields creates the Fields collection from C data
func (c *cgoImpl) buildFields() error {
	if c.data == nil {
		return fmt.Errorf("no database open")
	}

	// Get field count from codebase
	fieldCount := int(C.d4numFields(c.data))
	if fieldCount <= 0 {
		return fmt.Errorf("no fields found in database")
	}

	fields := make([]Field, fieldCount)
	indices := make(map[string]int)

	// Read each field definition
	for i := 0; i < fieldCount; i++ {
		// Get field pointer from codebase (1-indexed in C)
		cField := C.d4fieldJ(c.data, C.int(i+1))
		if cField == nil {
			return fmt.Errorf("failed to get field %d", i+1)
		}

		// Create foxi field wrapper
		field := &cgoField{
			impl:   c,
			cField: cField,
		}

		fields[i] = field

		// Create case-insensitive name mapping
		name := strings.ToLower(field.Name())
		indices[name] = i
	}

	c.fields = &Fields{
		fields:  fields,
		indices: indices,
	}

	return nil
}

// cgoField implements the Field interface using C library
type cgoField struct {
	impl   *cgoImpl
	cField *C.FIELD4
}

// Value returns the field's native value
func (f *cgoField) Value() (interface{}, error) {
	if f.impl.data == nil {
		return nil, fmt.Errorf("database not open")
	}

	// Get field value from current record using C library
	fieldPtr := C.f4ptr(f.cField)
	if fieldPtr == nil {
		return nil, fmt.Errorf("failed to get field pointer")
	}

	// Convert based on field type
	fieldType := rune(f.cField._type)
	switch fieldType {
	case 'C':
		return C.GoString((*C.char)(fieldPtr)), nil
	case 'N', 'F':
		return C.f4double(f.cField), nil
	case 'L':
		return C.f4true(f.cField) != 0, nil
	case 'D':
		return C.GoString((*C.char)(fieldPtr)), nil
	case 'I':
		return int(C.f4long(f.cField)), nil
	default:
		return C.GoString((*C.char)(fieldPtr)), nil
	}
}

// AsString returns field value as string
func (f *cgoField) AsString() (string, error) {
	if f.impl.data == nil {
		return "", fmt.Errorf("database not open")
	}

	fieldPtr := C.f4ptr(f.cField)
	if fieldPtr == nil {
		return "", fmt.Errorf("failed to get field pointer")
	}

	return C.GoString((*C.char)(fieldPtr)), nil
}

// AsInt returns field value as integer
func (f *cgoField) AsInt() (int, error) {
	if f.impl.data == nil {
		return 0, fmt.Errorf("database not open")
	}

	return int(C.f4long(f.cField)), nil
}

// AsFloat returns field value as float64
func (f *cgoField) AsFloat() (float64, error) {
	if f.impl.data == nil {
		return 0, fmt.Errorf("database not open")
	}

	return float64(C.f4double(f.cField)), nil
}

// AsBool returns field value as boolean
func (f *cgoField) AsBool() (bool, error) {
	if f.impl.data == nil {
		return false, fmt.Errorf("database not open")
	}

	return C.f4true(f.cField) != 0, nil
}

// AsTime returns field value as time.Time
func (f *cgoField) AsTime() (time.Time, error) {
	if f.impl.data == nil {
		return time.Time{}, fmt.Errorf("database not open")
	}

	// Convert from C date format
	fieldPtr := C.f4ptr(f.cField)
	if fieldPtr == nil {
		return time.Time{}, fmt.Errorf("failed to get field pointer")
	}

	dateStr := C.GoString((*C.char)(fieldPtr))
	if len(dateStr) != 8 {
		return time.Time{}, fmt.Errorf("invalid date format")
	}

	return time.Parse("20060102", dateStr)
}

// IsNull checks if field value is null
func (f *cgoField) IsNull() (bool, error) {
	if f.impl.data == nil {
		return false, fmt.Errorf("database not open")
	}

	// Check for null using C library
	return C.f4null(f.cField) != 0, nil
}

// Name returns field name
func (f *cgoField) Name() string {
	return C.GoString(&f.cField.name[0])
}

// Type returns field type
func (f *cgoField) Type() FieldType {
	cType := rune(f.cField._type)
	return convertFromCFieldType(cType)
}

// Size returns field size
func (f *cgoField) Size() uint8 {
	return uint8(f.cField.len)
}

// Decimals returns decimal places
func (f *cgoField) Decimals() uint8 {
	return uint8(f.cField.dec)
}

// IsSystem returns if field is system field
func (f *cgoField) IsSystem() bool {
	// Basic implementation - could be enhanced
	return false
}

// IsNullable returns if field can be null
func (f *cgoField) IsNullable() bool {
	return f.cField.null != 0
}

// IsBinary returns if field contains binary data
func (f *cgoField) IsBinary() bool {
	return f.cField.binary != 0
}

// ==========================================================================
// CGO FIELD MUST VARIANTS - Panic instead of returning errors
// ==========================================================================

// MustValue returns the field's native value, panicking on error
func (f *cgoField) MustValue() interface{} {
	value, err := f.Value()
	if err != nil {
		panic(err)
	}
	return value
}

// MustAsString returns field value as string, panicking on error
func (f *cgoField) MustAsString() string {
	value, err := f.AsString()
	if err != nil {
		panic(err)
	}
	return value
}

// MustAsInt returns field value as integer, panicking on error
func (f *cgoField) MustAsInt() int {
	value, err := f.AsInt()
	if err != nil {
		panic(err)
	}
	return value
}

// MustAsFloat returns field value as float64, panicking on error
func (f *cgoField) MustAsFloat() float64 {
	value, err := f.AsFloat()
	if err != nil {
		panic(err)
	}
	return value
}

// MustAsBool returns field value as boolean, panicking on error
func (f *cgoField) MustAsBool() bool {
	value, err := f.AsBool()
	if err != nil {
		panic(err)
	}
	return value
}

// MustAsTime returns field value as time.Time, panicking on error
func (f *cgoField) MustAsTime() time.Time {
	value, err := f.AsTime()
	if err != nil {
		panic(err)
	}
	return value
}

// MustIsNull checks if field value is null, panicking on error
func (f *cgoField) MustIsNull() bool {
	value, err := f.IsNull()
	if err != nil {
		panic(err)
	}
	return value
}

// convertFromCFieldType converts C field type to foxi FieldType
func convertFromCFieldType(cType rune) FieldType {
	switch cType {
	case 'C':
		return FTCharacter
	case 'N':
		return FTNumeric
	case 'L':
		return FTLogical
	case 'D':
		return FTDate
	case 'I':
		return FTInteger
	case 'T':
		return FTDateTime
	case 'Y':
		return FTCurrency
	case 'M':
		return FTMemo
	case 'B':
		return FTBlob
	case 'F':
		return FTFloat
	case 'G':
		return FTGeneral
	case 'P':
		return FTPicture
	case 'Q':
		return FTVarBinary
	case 'V':
		return FTVarchar
	case 'W':
		return FTTimestamp
	case 'X':
		return FTDouble
	default:
		return FTUnknown
	}
}

// =========================================================================
// CGO INDEX IMPLEMENTATION
// =========================================================================

// cgoIndexesImpl implements indexesImpl for the CGO backend
type cgoIndexesImpl struct {
	data    *C.DATA4
	indexes []Index
	tags    []Tag
	loaded  bool
}

// Load discovers and loads all available indexes
func (idx *cgoIndexesImpl) Load() error {
	if idx.data == nil {
		return fmt.Errorf("database not open")
	}

	if idx.loaded {
		return nil // Already loaded
	}

	var indexes []Index
	var allTags []Tag

	// Try to open production index (same name as DBF with .CDX extension)
	if idx.data.dataFile != nil {
		dbfFileName := C.GoString(&idx.data.dataFile.file.name[0])
		if dbfFileName != "" {
			baseName := strings.TrimSuffix(dbfFileName, ".dbf")
			cdxFileName := baseName + ".cdx"

			// Convert to C string
			cCdxFileName := C.CString(cdxFileName)
			defer C.free(unsafe.Pointer(cCdxFileName))

			// Attempt to open the production index
			index4 := C.i4open(idx.data, cCdxFileName)
			if index4 != nil {
				index := &cgoIndex{
					index4:       index4,
					data:         idx.data,
					isProduction: true,
				}
				indexes = append(indexes, index)

				// Load tags from this index
				indexTags := index.Tags()
				allTags = append(allTags, indexTags...)
			}
		}
	}

	// TODO: Look for additional standalone index files
	// This would require scanning the directory for additional index files

	idx.indexes = indexes
	idx.tags = allTags
	idx.loaded = true

	return nil
}

// Count returns the number of loaded indexes
func (idx *cgoIndexesImpl) Count() int {
	return len(idx.indexes)
}

// Loaded returns true if indexes have been loaded
func (idx *cgoIndexesImpl) Loaded() bool {
	return idx.loaded
}

// ByIndex returns the index at the specified position
func (idx *cgoIndexesImpl) ByIndex(index int) Index {
	if index < 0 || index >= len(idx.indexes) {
		return nil
	}
	return idx.indexes[index]
}

// ByName returns the index with the specified name
func (idx *cgoIndexesImpl) ByName(name string) Index {
	for _, index := range idx.indexes {
		if strings.EqualFold(index.Name(), name) {
			return index
		}
	}
	return nil
}

// List returns all indexes
func (idx *cgoIndexesImpl) List() []Index {
	return idx.indexes
}

// TagByName returns the tag with the specified name
func (idx *cgoIndexesImpl) TagByName(name string) Tag {
	for _, tag := range idx.tags {
		if strings.EqualFold(tag.Name(), name) {
			return tag
		}
	}
	return nil
}

// SelectedTag returns the currently selected tag
func (idx *cgoIndexesImpl) SelectedTag() Tag {
	if idx.data == nil {
		return nil
	}

	selectedTag := C.d4tag(idx.data, nil) // Get current tag
	if selectedTag == nil {
		return nil
	}

	// Find the matching foxi tag
	for _, tag := range idx.tags {
		if cgoTag, ok := tag.(*cgoTag); ok {
			if cgoTag.tag4 == selectedTag {
				return tag
			}
		}
	}

	return nil
}

// SelectTag sets the active tag
func (idx *cgoIndexesImpl) SelectTag(tag Tag) error {
	if idx.data == nil {
		return fmt.Errorf("database not open")
	}

	if tag == nil {
		// Select natural order (no index)
		C.d4tagSelect(idx.data, nil)
		return nil
	}

	if cgoTag, ok := tag.(*cgoTag); ok {
		C.d4tagSelect(idx.data, cgoTag.tag4)
		return nil
	}

	return fmt.Errorf("invalid tag type")
}

// Tags returns all available tags
func (idx *cgoIndexesImpl) Tags() []Tag {
	return idx.tags
}

// cgoIndex implements Index for the CGO backend
type cgoIndex struct {
	index4       *C.INDEX4
	data         *C.DATA4
	tags         []Tag
	isProduction bool
}

// Name returns the index name
func (idx *cgoIndex) Name() string {
	if idx.index4 == nil {
		return ""
	}
	fileName := C.i4fileName(idx.index4)
	if fileName == nil {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(C.GoString(fileName)), ".cdx")
}

// FileName returns the index file name
func (idx *cgoIndex) FileName() string {
	if idx.index4 == nil {
		return ""
	}
	fileName := C.i4fileName(idx.index4)
	if fileName == nil {
		return ""
	}
	return C.GoString(fileName)
}

// TagCount returns the number of tags in this index
func (idx *cgoIndex) TagCount() int {
	if idx.tags == nil {
		idx.loadTags()
	}
	return len(idx.tags)
}

// Tag returns the tag at the specified index
func (idx *cgoIndex) Tag(index int) Tag {
	if idx.tags == nil {
		idx.loadTags()
	}
	if index < 0 || index >= len(idx.tags) {
		return nil
	}
	return idx.tags[index]
}

// TagByName returns the tag with the specified name
func (idx *cgoIndex) TagByName(name string) Tag {
	if idx.tags == nil {
		idx.loadTags()
	}
	for _, tag := range idx.tags {
		if strings.EqualFold(tag.Name(), name) {
			return tag
		}
	}
	return nil
}

// Tags returns all tags in this index
func (idx *cgoIndex) Tags() []Tag {
	if idx.tags == nil {
		idx.loadTags()
	}
	return idx.tags
}

// IsOpen returns true if the index is open
func (idx *cgoIndex) IsOpen() bool {
	return idx.index4 != nil
}

// IsProduction returns true if this is the production index
func (idx *cgoIndex) IsProduction() bool {
	return idx.isProduction
}

// loadTags loads all tags from this index
func (idx *cgoIndex) loadTags() {
	if idx.index4 == nil || idx.data == nil {
		return
	}

	var tags []Tag

	// Iterate through all tags in the index using C library
	// Start with first tag in index
	tag4 := C.i4tag(idx.index4, nil) // Get first tag
	for tag4 != nil {
		tag := &cgoTag{
			tag4:  tag4,
			data:  idx.data,
			index: idx,
		}
		tags = append(tags, tag)

		// Get next tag (this is a simplified approach)
		// In reality, we'd need to iterate through the tag list properly
		break // For now, just get the first one
	}

	idx.tags = tags
}

// cgoTag implements Tag for the CGO backend
type cgoTag struct {
	tag4  *C.TAG4
	data  *C.DATA4
	index *cgoIndex
}

// Name returns the tag name
func (tag *cgoTag) Name() string {
	if tag.tag4 == nil {
		return ""
	}
	alias := C.t4alias(tag.tag4)
	if alias == nil {
		return ""
	}
	return C.GoString(alias)
}

// Expression returns the tag expression
func (tag *cgoTag) Expression() string {
	if tag.tag4 == nil {
		return ""
	}
	// Use the t4expr macro equivalent
	if tag.tag4.tagFile != nil && tag.tag4.tagFile.expr != nil {
		return C.GoString(tag.tag4.tagFile.expr.source)
	}
	return ""
}

// Filter returns the tag filter expression
func (tag *cgoTag) Filter() string {
	if tag.tag4 == nil {
		return ""
	}
	// Use the t4filter macro equivalent
	if tag.tag4.tagFile != nil && tag.tag4.tagFile.filter != nil {
		return C.GoString(tag.tag4.tagFile.filter.source)
	}
	return ""
}

// KeyLength returns the key length
func (tag *cgoTag) KeyLength() int {
	if tag.tag4 == nil || tag.tag4.tagFile == nil {
		return 0
	}
	return int(tag.tag4.tagFile.header.keyLen)
}

// IsUnique returns true if the tag enforces uniqueness
func (tag *cgoTag) IsUnique() bool {
	if tag.tag4 == nil {
		return false
	}
	return C.t4unique(tag.tag4) != 0
}

// IsDescending returns true if the tag is in descending order
func (tag *cgoTag) IsDescending() bool {
	if tag.tag4 == nil || tag.tag4.tagFile == nil {
		return false
	}
	return tag.tag4.tagFile.header.descending != 0
}

// IsSelected returns true if this tag is currently selected
func (tag *cgoTag) IsSelected() bool {
	if tag.data == nil {
		return false
	}
	currentTag := C.d4tag(tag.data, nil)
	return currentTag == tag.tag4
}

// Seek performs a seek operation with generic value
func (tag *cgoTag) Seek(value interface{}) (SeekResult, error) {
	if tag.data == nil {
		return SeekEOF, fmt.Errorf("database not open")
	}

	// Convert value to string for seeking
	var searchValue string
	switch v := value.(type) {
	case string:
		searchValue = v
	case int:
		searchValue = fmt.Sprintf("%d", v)
	case float64:
		searchValue = fmt.Sprintf("%g", v)
	default:
		searchValue = fmt.Sprintf("%v", v)
	}

	return tag.SeekString(searchValue)
}

// SeekString performs a seek operation with string value
func (tag *cgoTag) SeekString(value string) (SeekResult, error) {
	if tag.data == nil || tag.tag4 == nil {
		return SeekEOF, fmt.Errorf("database not open")
	}

	// Select this tag first
	C.d4tagSelect(tag.data, tag.tag4)

	// Convert Go string to C string
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	// Perform seek using C library
	result := C.tfile4seek(tag.tag4.tagFile, unsafe.Pointer(cValue), C.int(len(value)))

	// Convert C result to foxi result
	switch int(result) {
	case 0: // r4success
		return SeekSuccess, nil
	case 1: // r4after
		return SeekAfter, nil
	case 2: // r4eof
		return SeekEOF, nil
	default:
		return SeekEOF, fmt.Errorf("seek failed with code %d", int(result))
	}
}

// SeekDouble performs a seek operation with float64 value
func (tag *cgoTag) SeekDouble(value float64) (SeekResult, error) {
	return tag.SeekString(fmt.Sprintf("%g", value))
}

// SeekInt performs a seek operation with int value
func (tag *cgoTag) SeekInt(value int) (SeekResult, error) {
	return tag.SeekString(fmt.Sprintf("%d", value))
}

// First moves to first record in tag order
func (tag *cgoTag) First() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Select this tag and go to first
	C.d4tagSelect(tag.data, tag.tag4)
	result := C.tfile4top(tag.tag4.tagFile)
	if result != 0 {
		return fmt.Errorf("failed to go to first record")
	}
	return nil
}

// Last moves to last record in tag order
func (tag *cgoTag) Last() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Select this tag and go to last
	C.d4tagSelect(tag.data, tag.tag4)
	result := C.tfile4bottom(tag.tag4.tagFile)
	if result != 0 {
		return fmt.Errorf("failed to go to last record")
	}
	return nil
}

// Next moves to next record in tag order
func (tag *cgoTag) Next() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	result := C.tfile4skip(tag.tag4.tagFile, 1)
	if result == 0 {
		return fmt.Errorf("failed to move to next record")
	}
	return nil
}

// Previous moves to previous record in tag order
func (tag *cgoTag) Previous() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	result := C.tfile4skip(tag.tag4.tagFile, -1)
	if result == 0 {
		return fmt.Errorf("failed to move to previous record")
	}
	return nil
}

// Position returns the current position as a percentage (0.0-1.0)
func (tag *cgoTag) Position() float64 {
	if tag.data == nil || tag.tag4 == nil {
		return 0.0
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	return float64(C.tfile4position(tag.tag4.tagFile))
}

// PositionSet moves to the specified position percentage (0.0-1.0)
func (tag *cgoTag) PositionSet(percent float64) error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	result := C.tfile4positionSet(tag.tag4.tagFile, C.double(percent))
	if result != 0 {
		return fmt.Errorf("failed to set position")
	}
	return nil
}

// CurrentKey returns the current index key value
func (tag *cgoTag) CurrentKey() string {
	if tag.data == nil || tag.tag4 == nil {
		return ""
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	keyPtr := C.tfile4key(tag.tag4.tagFile)
	if keyPtr == nil {
		return ""
	}
	return C.GoString(keyPtr)
}

// RecordNumber returns the current record number
func (tag *cgoTag) RecordNumber() int {
	if tag.data == nil || tag.tag4 == nil {
		return 0
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	return int(C.tfile4recNo(tag.tag4.tagFile))
}

// EOF returns true if at end of index
func (tag *cgoTag) EOF() bool {
	if tag.data == nil || tag.tag4 == nil {
		return true
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	return C.tfile4eof(tag.tag4.tagFile) != 0
}

// BOF returns true if at beginning of index
func (tag *cgoTag) BOF() bool {
	if tag.data == nil || tag.tag4 == nil {
		return true
	}

	// Ensure this tag is selected
	currentTag := C.d4tag(tag.data, nil)
	if currentTag != tag.tag4 {
		C.d4tagSelect(tag.data, tag.tag4)
	}

	// Note: C library doesn't have tfile4bof, check if at first record
	pos := C.tfile4position(tag.tag4.tagFile)
	return pos <= 0.0
}

// ==========================================================================
// CGO TAG MUST VARIANTS - Panic instead of returning errors
// ==========================================================================

// MustSeek performs a seek operation with generic value, panicking on error
func (tag *cgoTag) MustSeek(value interface{}) SeekResult {
	result, err := tag.Seek(value)
	if err != nil {
		panic(err)
	}
	return result
}

// MustSeekString performs a seek operation with string value, panicking on error
func (tag *cgoTag) MustSeekString(value string) SeekResult {
	result, err := tag.SeekString(value)
	if err != nil {
		panic(err)
	}
	return result
}

// MustSeekDouble performs a seek operation with float64 value, panicking on error
func (tag *cgoTag) MustSeekDouble(value float64) SeekResult {
	result, err := tag.SeekDouble(value)
	if err != nil {
		panic(err)
	}
	return result
}

// MustSeekInt performs a seek operation with int value, panicking on error
func (tag *cgoTag) MustSeekInt(value int) SeekResult {
	result, err := tag.SeekInt(value)
	if err != nil {
		panic(err)
	}
	return result
}

// MustFirst moves to first record in tag order, panicking on error
func (tag *cgoTag) MustFirst() {
	if err := tag.First(); err != nil {
		panic(err)
	}
}

// MustLast moves to last record in tag order, panicking on error
func (tag *cgoTag) MustLast() {
	if err := tag.Last(); err != nil {
		panic(err)
	}
}

// MustNext moves to next record in tag order, panicking on error
func (tag *cgoTag) MustNext() {
	if err := tag.Next(); err != nil {
		panic(err)
	}
}

// MustPrevious moves to previous record in tag order, panicking on error
func (tag *cgoTag) MustPrevious() {
	if err := tag.Previous(); err != nil {
		panic(err)
	}
}

// MustPositionSet moves to the specified position percentage, panicking on error
func (tag *cgoTag) MustPositionSet(percent float64) {
	if err := tag.PositionSet(percent); err != nil {
		panic(err)
	}
}
