//go:build !foxicgo
// +build !foxicgo

package foxi

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pkg "github.com/mkfoss/foxi/pkg/gocore"
)

// pureGoImpl implements foxiImpl using the gomkfdbf pure Go backend
type pureGoImpl struct {
	codeBase *pkg.Code4
	data     *pkg.Data4
	fields   *Fields
	indexes  *Indexes
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

// Indexes returns the index collection
func (p *pureGoImpl) Indexes() *Indexes {
	if p.indexes == nil {
		p.indexes = &Indexes{
			impl: &pureGoIndexesImpl{
				data:   p.data,
				loaded: false,
			},
		}
	}
	return p.indexes
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

// =========================================================================
// PURE GO INDEX IMPLEMENTATION
// =========================================================================

// pureGoIndexesImpl implements indexesImpl for the pure Go backend
type pureGoIndexesImpl struct {
	data    *pkg.Data4
	indexes []Index
	tags    []Tag
	loaded  bool
}

// Load discovers and loads all available indexes
func (idx *pureGoIndexesImpl) Load() error {
	if idx.data == nil {
		return fmt.Errorf("database not open")
	}

	if idx.loaded {
		return nil // Already loaded
	}

	var indexes []Index
	var allTags []Tag

	// Try to open production index (same name as DBF with .CDX extension)
	dbfFileName := pkg.D4FileName(idx.data)
	if dbfFileName != "" {
		baseName := strings.TrimSuffix(dbfFileName, ".dbf")
		cdxFileName := baseName + ".cdx"
		
		// Attempt to open the production index
		index4 := pkg.I4Open(idx.data, cdxFileName)
		if index4 != nil {
			index := &pureGoIndex{
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

	// TODO: Look for additional standalone index files (.IDX, named .CDX files)
	// This would require scanning the directory for additional index files

	idx.indexes = indexes
	idx.tags = allTags
	idx.loaded = true

	return nil
}

// Count returns the number of loaded indexes
func (idx *pureGoIndexesImpl) Count() int {
	return len(idx.indexes)
}

// Loaded returns true if indexes have been loaded
func (idx *pureGoIndexesImpl) Loaded() bool {
	return idx.loaded
}

// ByIndex returns the index at the specified position
func (idx *pureGoIndexesImpl) ByIndex(index int) Index {
	if index < 0 || index >= len(idx.indexes) {
		return nil
	}
	return idx.indexes[index]
}

// ByName returns the index with the specified name
func (idx *pureGoIndexesImpl) ByName(name string) Index {
	for _, index := range idx.indexes {
		if strings.EqualFold(index.Name(), name) {
			return index
		}
	}
	return nil
}

// List returns all indexes
func (idx *pureGoIndexesImpl) List() []Index {
	return idx.indexes
}

// TagByName returns the tag with the specified name
func (idx *pureGoIndexesImpl) TagByName(name string) Tag {
	for _, tag := range idx.tags {
		if strings.EqualFold(tag.Name(), name) {
			return tag
		}
	}
	return nil
}

// SelectedTag returns the currently selected tag
func (idx *pureGoIndexesImpl) SelectedTag() Tag {
	if idx.data == nil {
		return nil
	}
	
	selectedTag := pkg.D4TagSelected(idx.data)
	if selectedTag == nil {
		return nil
	}
	
	// Find the matching foxi tag
	for _, tag := range idx.tags {
		if pureTag, ok := tag.(*pureGoTag); ok {
			if pureTag.tag4 == selectedTag {
				return tag
			}
		}
	}
	
	return nil
}

// SelectTag sets the active tag
func (idx *pureGoIndexesImpl) SelectTag(tag Tag) error {
	if idx.data == nil {
		return fmt.Errorf("database not open")
	}
	
	if tag == nil {
		// Select natural order (no index)
		pkg.D4TagSelect(idx.data, nil)
		return nil
	}
	
	if pureTag, ok := tag.(*pureGoTag); ok {
		pkg.D4TagSelect(idx.data, pureTag.tag4)
		return nil
	}
	
	return fmt.Errorf("invalid tag type")
}

// Tags returns all available tags
func (idx *pureGoIndexesImpl) Tags() []Tag {
	return idx.tags
}

// pureGoIndex implements Index for the pure Go backend
type pureGoIndex struct {
	index4       *pkg.Index4
	data         *pkg.Data4
	tags         []Tag
	isProduction bool
}

// Name returns the index name
func (idx *pureGoIndex) Name() string {
	if idx.index4 == nil {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(string(idx.index4.AccessName[:])), ".cdx")
}

// FileName returns the index file name
func (idx *pureGoIndex) FileName() string {
	if idx.index4 == nil {
		return ""
	}
	return string(idx.index4.AccessName[:])
}

// TagCount returns the number of tags in this index
func (idx *pureGoIndex) TagCount() int {
	if idx.index4 == nil {
		return 0
	}
	return pkg.I4NumTags(idx.index4)
}

// Tag returns the tag at the specified index
func (idx *pureGoIndex) Tag(index int) Tag {
	if idx.tags == nil {
		idx.loadTags()
	}
	if index < 0 || index >= len(idx.tags) {
		return nil
	}
	return idx.tags[index]
}

// TagByName returns the tag with the specified name
func (idx *pureGoIndex) TagByName(name string) Tag {
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
func (idx *pureGoIndex) Tags() []Tag {
	if idx.tags == nil {
		idx.loadTags()
	}
	return idx.tags
}

// IsOpen returns true if the index is open
func (idx *pureGoIndex) IsOpen() bool {
	return idx.index4 != nil && idx.index4.IsValid
}

// IsProduction returns true if this is the production index
func (idx *pureGoIndex) IsProduction() bool {
	return idx.isProduction
}

// loadTags loads all tags from this index
func (idx *pureGoIndex) loadTags() {
	if idx.index4 == nil || idx.data == nil {
		return
	}

	var tags []Tag

	// Iterate through all tags in this index using gomkfdbf
	var tag4 *pkg.Tag4 = nil
	for {
		tag4 = pkg.D4TagNext(idx.data, tag4)
		if tag4 == nil {
			break
		}
		
		// Check if this tag belongs to our index
		if tag4.Index == idx.index4 {
			tag := &pureGoTag{
				tag4:  tag4,
				data:  idx.data,
				index: idx,
			}
			tags = append(tags, tag)
		}
	}

	idx.tags = tags
}

// pureGoTag implements Tag for the pure Go backend
type pureGoTag struct {
	tag4  *pkg.Tag4
	data  *pkg.Data4
	index *pureGoIndex
}

// Name returns the tag name
func (tag *pureGoTag) Name() string {
	return pkg.T4Name(tag.tag4)
}

// Expression returns the tag expression
func (tag *pureGoTag) Expression() string {
	return pkg.T4Expr(tag.tag4)
}

// Filter returns the tag filter expression
func (tag *pureGoTag) Filter() string {
	// gomkfdbf doesn't expose filter directly in current API
	return ""
}

// KeyLength returns the key length
func (tag *pureGoTag) KeyLength() int {
	return int(pkg.T4KeyLen(tag.tag4))
}

// IsUnique returns true if the tag enforces uniqueness
func (tag *pureGoTag) IsUnique() bool {
	return pkg.T4Unique(tag.tag4)
}

// IsDescending returns true if the tag is in descending order
func (tag *pureGoTag) IsDescending() bool {
	return pkg.T4Descending(tag.tag4)
}

// IsSelected returns true if this tag is currently selected
func (tag *pureGoTag) IsSelected() bool {
	if tag.data == nil {
		return false
	}
	return pkg.D4TagSelected(tag.data) == tag.tag4
}

// Seek performs a seek operation with generic value
func (tag *pureGoTag) Seek(value interface{}) (SeekResult, error) {
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
func (tag *pureGoTag) SeekString(value string) (SeekResult, error) {
	if tag.data == nil || tag.tag4 == nil {
		return SeekEOF, fmt.Errorf("database not open")
	}

	// Select this tag first
	pkg.D4TagSelect(tag.data, tag.tag4)

	// Perform seek using gomkfdbf D4Seek function
	result := pkg.D4Seek(tag.data, value)

	// Convert gomkfdbf result to foxi result
	switch result {
	case pkg.R4Success:
		return SeekSuccess, nil
	case pkg.R4After:
		return SeekAfter, nil
	case pkg.R4Eof:
		return SeekEOF, nil
	default:
		return SeekEOF, fmt.Errorf("seek failed with code %d", result)
	}
}

// SeekDouble performs a seek operation with float64 value
func (tag *pureGoTag) SeekDouble(value float64) (SeekResult, error) {
	return tag.SeekString(fmt.Sprintf("%g", value))
}

// SeekInt performs a seek operation with int value
func (tag *pureGoTag) SeekInt(value int) (SeekResult, error) {
	return tag.SeekString(fmt.Sprintf("%d", value))
}

// First moves to first record in tag order
func (tag *pureGoTag) First() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Select this tag and go to first using data-level navigation
	pkg.D4TagSelect(tag.data, tag.tag4)
	result := pkg.D4Top(tag.data)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to go to first record")
	}
	return nil
}

// Last moves to last record in tag order
func (tag *pureGoTag) Last() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Select this tag and go to last using data-level navigation
	pkg.D4TagSelect(tag.data, tag.tag4)
	result := pkg.D4Bottom(tag.data)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to go to last record")
	}
	return nil
}

// Next moves to next record in tag order
func (tag *pureGoTag) Next() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Ensure this tag is selected
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	result := pkg.D4Skip(tag.data, 1)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to move to next record")
	}
	return nil
}

// Previous moves to previous record in tag order
func (tag *pureGoTag) Previous() error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Ensure this tag is selected
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	result := pkg.D4Skip(tag.data, -1)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to move to previous record")
	}
	return nil
}

// Position returns the current position as a percentage (0.0-1.0)
func (tag *pureGoTag) Position() float64 {
	if tag.data == nil || tag.tag4 == nil {
		return 0.0
	}

	// Ensure this tag is selected
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	// Simple position calculation based on record number
	current := float64(pkg.D4RecNo(tag.data))
	total := float64(tag.data.DataFile.Header.NumRecs)
	if total <= 0 {
		return 0.0
	}
	return current / total
}

// PositionSet moves to the specified position percentage (0.0-1.0)
func (tag *pureGoTag) PositionSet(percent float64) error {
	if tag.data == nil || tag.tag4 == nil {
		return fmt.Errorf("database not open")
	}

	// Ensure this tag is selected
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	// Calculate record number from percentage
	total := int32(tag.data.DataFile.Header.NumRecs)
	recordNo := int32(percent * float64(total))
	if recordNo < 1 {
		recordNo = 1
	}
	if recordNo > total {
		recordNo = total
	}

	result := pkg.D4Go(tag.data, recordNo)
	if result != pkg.ErrorNone {
		return fmt.Errorf("failed to set position")
	}
	return nil
}

// CurrentKey returns the current index key value
func (tag *pureGoTag) CurrentKey() string {
	if tag.data == nil || tag.tag4 == nil {
		return ""
	}

	// Ensure this tag is selected
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	// For now, return a simple representation
	// In a full implementation, this would evaluate the tag expression
	return fmt.Sprintf("key_%d", pkg.D4RecNo(tag.data))
}

// RecordNumber returns the current record number
func (tag *pureGoTag) RecordNumber() int {
	if tag.data == nil || tag.tag4 == nil {
		return 0
	}

	// Ensure this tag is selected
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	return int(pkg.D4RecNo(tag.data))
}

// EOF returns true if at end of index
func (tag *pureGoTag) EOF() bool {
	if tag.data == nil || tag.tag4 == nil {
		return true
	}

	// Ensure this tag is selected
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	return pkg.D4Eof(tag.data)
}

// BOF returns true if at beginning of index
func (tag *pureGoTag) BOF() bool {
	if tag.data == nil || tag.tag4 == nil {
		return true
	}

	// Ensure this tag is selected  
	if pkg.D4TagSelected(tag.data) != tag.tag4 {
		pkg.D4TagSelect(tag.data, tag.tag4)
	}

	return pkg.D4Bof(tag.data)
}
