//go:build !foxicgo
// +build !foxicgo

package foxi

import (
	"fmt"
	"strings"
	"time"

	pkg "github.com/mkfoss/foxi/pkg/gocore"
)

// pureGoImpl implements foxiImpl using the gomkfdbf pure Go backend
type pureGoImpl struct {
	codeBase *pkg.Code4
	data     *pkg.Data4
	fields   *Fields
	filename string
}

// init function creates the implementation instance when package loads
func init() {
	// This will be called when the package is imported without foxicgo tag
}

// NewFoxi creates a new Foxi instance with pure Go backend
func NewFoxi() *Foxi {
	impl := &pureGoImpl{}
	return &Foxi{impl: impl}
}

// Open establishes a connection to the specified DBF file using gomkfdbf
func (p *pureGoImpl) Open(filename string) error {
	if p.data != nil {
		return fmt.Errorf("database already open")
	}

	// Initialize CODE4 structure
	p.codeBase = &pkg.Code4{}
	
	// Set default configuration
	p.codeBase.AutoOpen = true
	p.codeBase.ErrOff = 0 // Show errors
	
	// Open the data file using gomkfdbf
	p.data = pkg.D4Open(p.codeBase, filename)
	if p.data == nil {
		return fmt.Errorf("failed to open database file: %s", filename)
	}

	p.filename = filename

	// Build Fields collection from gomkfdbf data
	err := p.buildFields()
	if err != nil {
		p.Close()
		return err
	}

	return nil
}

// Close closes the database connection and releases resources
func (p *pureGoImpl) Close() error {
	if p.data == nil {
		return nil
	}

	// Close data in gomkfdbf
	pkg.D4Close(p.data)
	p.data = nil
	p.codeBase = nil
	p.fields = nil
	p.filename = ""

	return nil
}

// Active reports whether the database connection is active
func (p *pureGoImpl) Active() bool {
	return p.data != nil
}

// Header returns database header information
func (p *pureGoImpl) Header() Header {
	if p.data == nil {
		return Header{}
	}

	// Get header information from gomkfdbf
	dataFile := p.data.DataFile
	if dataFile == nil {
		return Header{}
	}

	header := Header{
		recordCount: uint(dataFile.Header.NumRecs),
		hasIndex:    false, // Will be set based on actual index detection
		hasFpt:      false, // Will be set based on memo field detection
		codepage:    Codepage(0x03), // Default to Windows ANSI
	}

	// Parse last updated date
	year := int(dataFile.Header.Year)
	if year < 80 {
		year += 2000 // Y2K handling
	} else {
		year += 1900
	}
	
	month := int(dataFile.Header.Month)
	day := int(dataFile.Header.Day)
	
	if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
		header.lastUpdated = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	return header
}

// Fields returns the field collection
func (p *pureGoImpl) Fields() *Fields {
	return p.fields
}

// FieldCount returns the number of fields
func (p *pureGoImpl) FieldCount() int {
	if p.fields == nil {
		return 0
	}
	return p.fields.Count()
}

// Field returns field by index
func (p *pureGoImpl) Field(index int) Field {
	if p.fields == nil {
		return nil
	}
	return p.fields.ByIndex(index)
}

// FieldByName returns field by name
func (p *pureGoImpl) FieldByName(name string) Field {
	if p.fields == nil {
		return nil
	}
	return p.fields.ByName(name)
}

// Navigation methods
func (p *pureGoImpl) Goto(recordNumber int) error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	result := pkg.D4Go(p.data, int32(recordNumber))
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to goto record %d", recordNumber)
	}
	return nil
}

func (p *pureGoImpl) First() error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	result := pkg.D4Top(p.data)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to go to first record")
	}
	return nil
}

func (p *pureGoImpl) Last() error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	result := pkg.D4Bottom(p.data)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to go to last record")
	}
	return nil
}

func (p *pureGoImpl) Next() error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	result := pkg.D4Skip(p.data, 1)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to move to next record")
	}
	return nil
}

func (p *pureGoImpl) Previous() error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	result := pkg.D4Skip(p.data, -1)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to move to previous record")
	}
	return nil
}

func (p *pureGoImpl) Skip(count int) error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	result := pkg.D4Skip(p.data, int32(count))
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to skip %d records", count)
	}
	return nil
}

func (p *pureGoImpl) Position() int {
	if p.data == nil {
		return 0
	}
	return int(pkg.D4RecNo(p.data))
}

func (p *pureGoImpl) EOF() bool {
	if p.data == nil {
		return true
	}
	return pkg.D4Eof(p.data)
}

func (p *pureGoImpl) BOF() bool {
	if p.data == nil {
		return true
	}
	return pkg.D4Bof(p.data)
}

// Record state methods
func (p *pureGoImpl) Deleted() bool {
	if p.data == nil {
		return false
	}
	return pkg.D4Deleted(p.data)
}

func (p *pureGoImpl) Delete() error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	pkg.D4Delete(p.data)
	return nil
}

func (p *pureGoImpl) Recall() error {
	if p.data == nil {
		return fmt.Errorf("database not open")
	}
	pkg.D4Recall(p.data)
	return nil
}

// Backend returns the backend type
func (p *pureGoImpl) Backend() Backend {
	return BackendPureGo
}

// buildFields creates the Fields collection from gomkfdbf data
func (p *pureGoImpl) buildFields() error {
	if p.data == nil || p.data.Fields == nil {
		return fmt.Errorf("no field data available")
	}

	fieldCount := len(p.data.Fields)
	fields := make([]Field, fieldCount)
	indices := make(map[string]int)

	for i, gomkField := range p.data.Fields {
		// Create foxi field wrapper
		field := &pureGoField{
			impl:     p,
			gomkField: gomkField,
		}
		
		fields[i] = field
		
		// Create case-insensitive name mapping
		name := strings.ToLower(string(gomkField.Name[:]))
		name = strings.TrimRight(name, "\x00") // Remove null terminators
		indices[name] = i
	}

	p.fields = &Fields{
		fields:  fields,
		indices: indices,
	}

	return nil
}

// pureGoField implements the Field interface using gomkfdbf
type pureGoField struct {
	impl      *pureGoImpl
	gomkField *pkg.Field4
}

// Value returns the field's native value
func (f *pureGoField) Value() (interface{}, error) {
	if f.impl.data == nil {
		return nil, fmt.Errorf("database not open")
	}
	
	// Get field value based on type
	switch rune(f.gomkField.Type) {
	case 'C':
		return pkg.F4Str(f.gomkField), nil
	case 'N', 'F':
		return pkg.F4Double(f.gomkField), nil
	case 'L':
		return pkg.F4True(f.gomkField), nil
	case 'I':
		return pkg.F4Long(f.gomkField), nil
	default:
		return pkg.F4Str(f.gomkField), nil
	}
}

// AsString returns field value as string
func (f *pureGoField) AsString() (string, error) {
	if f.impl.data == nil {
		return "", fmt.Errorf("database not open")
	}
	
	return pkg.F4Str(f.gomkField), nil
}

// AsInt returns field value as integer
func (f *pureGoField) AsInt() (int, error) {
	if f.impl.data == nil {
		return 0, fmt.Errorf("database not open")
	}
	
	return pkg.F4Int(f.gomkField), nil
}

// AsFloat returns field value as float64
func (f *pureGoField) AsFloat() (float64, error) {
	if f.impl.data == nil {
		return 0, fmt.Errorf("database not open")
	}
	
	return pkg.F4Double(f.gomkField), nil
}

// AsBool returns field value as boolean
func (f *pureGoField) AsBool() (bool, error) {
	if f.impl.data == nil {
		return false, fmt.Errorf("database not open")
	}
	
	return pkg.F4True(f.gomkField), nil
}

// AsTime returns field value as time.Time
func (f *pureGoField) AsTime() (time.Time, error) {
	if f.impl.data == nil {
		return time.Time{}, fmt.Errorf("database not open")
	}
	
	// Convert from gomkfdbf date format
	dateStr := pkg.F4Str(f.gomkField)
	if len(dateStr) != 8 {
		return time.Time{}, fmt.Errorf("invalid date format")
	}
	
	return time.Parse("20060102", dateStr)
}

// IsNull checks if field value is null
func (f *pureGoField) IsNull() (bool, error) {
	if f.impl.data == nil {
		return false, fmt.Errorf("database not open")
	}
	
	// Basic null checking - for now, check if field supports nulls
	// and if the field content is all spaces or empty
	if f.gomkField.Null == 0 {
		return false, nil // Field doesn't support nulls
	}
	
	// Check if field content is all spaces (DBF null representation)
	value := pkg.F4Str(f.gomkField)
	value = strings.TrimSpace(value)
	return len(value) == 0, nil
}

// Name returns field name
func (f *pureGoField) Name() string {
	name := string(f.gomkField.Name[:])
	return strings.TrimRight(name, "\x00") // Remove null terminators
}

// Type returns field type
func (f *pureGoField) Type() FieldType {
	gomkType := rune(f.gomkField.Type)
	return convertFromGomkFieldType(gomkType)
}

// Size returns field size
func (f *pureGoField) Size() uint8 {
	return uint8(f.gomkField.Length)
}

// Decimals returns decimal places
func (f *pureGoField) Decimals() uint8 {
	return uint8(f.gomkField.Dec)
}

// IsSystem returns if field is system field
func (f *pureGoField) IsSystem() bool {
	// Basic implementation - could be enhanced
	return false
}

// IsNullable returns if field can be null
func (f *pureGoField) IsNullable() bool {
	return f.gomkField.Null != 0
}

// IsBinary returns if field contains binary data
func (f *pureGoField) IsBinary() bool {
	return f.gomkField.Binary != 0
}

// convertFromGomkFieldType converts gomkfdbf field type to foxi FieldType
func convertFromGomkFieldType(gomkType rune) FieldType {
	switch gomkType {
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