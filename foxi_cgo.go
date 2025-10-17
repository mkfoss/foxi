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
		hasIndex:    false, // Will be detected based on actual indexes
		hasFpt:      false, // Will be detected based on memo fields
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