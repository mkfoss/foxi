// Package pkg - DATA4 functions
// Direct translation of CodeBase DBF operations
package pkg

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
)

// D4Open opens a Visual FoxPro DBF database file for reading and writing.
// This mirrors the d4open function from the CodeBase library.
//
// The function performs the following operations:
// - Opens the specified DBF file with .dbf extension if not provided
// - Parses the DBF header and validates the file format
// - Reads field definitions and creates field structures
// - Automatically opens associated memo files (.FPT) if memo fields exist
// - Auto-opens production indexes (.CDX) if AutoOpen is enabled
// - Adds the opened database to the CODE4 data file list
//
// Parameters:
//   - cb: CODE4 context for database operations and settings
//   - fileName: Path to the DBF file (extension optional)
//
// Returns a pointer to the initialized Data4 structure on success,
// nil on failure (error details available via CODE4 error handling).
func D4Open(cb *Code4, fileName string) *Data4 {
	if cb == nil || fileName == "" {
		setError(cb, ErrorMemory)
		return nil
	}

	// Construct full path with .dbf extension if needed
	fullPath := constructPath(fileName, "dbf")

	// Create DATA4 structure
	data := &Data4{
		CodeBase: cb,
		recNo:    0,
		atBof:    true,
		atEof:    false,
	}

	// Create DATA4FILE structure
	dataFile := &Data4File{
		CodeBase: cb,
	}

	// Open the DBF file
	err := File4Open(&dataFile.File, cb, fullPath, AccessDenyNone)
	if err != ErrorNone {
		return nil
	}

	// Read and parse DBF header
	err = parseDbfHeader(dataFile)
	if err != ErrorNone {
		File4Close(&dataFile.File)
		setError(cb, err)
		return nil
	}

	// Read field definitions
	err = parseFieldDefs(dataFile)
	if err != ErrorNone {
		File4Close(&dataFile.File)
		setError(cb, err)
		return nil
	}

	// Set up DATA4 structure
	data.DataFile = dataFile
	data.Fields = dataFile.Fields

	// Allocate record buffers
	recordLen := int(dataFile.RecordLen)
	data.Record = make([]byte, recordLen)
	data.RecordOld = make([]byte, recordLen)
	data.RecordBlank = make([]byte, recordLen)

	// Initialize blank record template
	initBlankRecord(data)

	// Auto-open memo file if database has memo fields
	if hasMemoFields(dataFile) {
		err = openMemoFile(dataFile, fileName)
		if err != ErrorNone {
			// Don't fail if memo file is missing, just log it
			// This allows databases with memo fields to open even without .FPT files
		}
	}

	// Auto-open index if enabled
	if cb.AutoOpen {
		autoOpenProductionIndex(data)
	}

	// Set alias from filename (base name without extension)
	alias := extractAlias(fileName)
	data.Alias = alias

	// Add to codebase data file list
	list4Add(&cb.DataFileList, &data.Link)

	setError(cb, ErrorNone)
	return data
}

// parseDbfHeader reads and parses the DBF file header
func parseDbfHeader(dataFile *Data4File) int {
	headerBuf := make([]byte, 32) // Basic header is 32 bytes

	// Read header
	bytesRead := File4Read(&dataFile.File, 0, headerBuf, 32)
	if bytesRead != 32 {
		return ErrorRead
	}

	header := &dataFile.Header

	// Parse basic header fields
	header.Version = headerBuf[0]
	header.Year = headerBuf[1]
	header.Month = headerBuf[2]
	header.Day = headerBuf[3]
	header.NumRecs = int32(binary.LittleEndian.Uint32(headerBuf[4:8]))
	header.HeaderLen = binary.LittleEndian.Uint16(headerBuf[8:10])
	header.RecordLen = binary.LittleEndian.Uint16(headerBuf[10:12])

	// Copy reserved area
	copy(header.Reserved[:], headerBuf[12:28])

	// Validate header
	if header.Version != 0x03 && header.Version != 0x30 && header.Version != 0x43 && header.Version != 0xF5 {
		return ErrorData // Unsupported DBF version
	}

	if header.HeaderLen < 33 || header.RecordLen < 1 {
		return ErrorData // Invalid header values
	}

	// Store record length in DATA4FILE
	dataFile.RecordLen = header.RecordLen

	return ErrorNone
}

// parseFieldDefs reads field definitions from DBF header
func parseFieldDefs(dataFile *Data4File) int {
	// Read field definitions until we hit the header terminator (0x0D)
	// Start after the 32-byte main header
	var fields []*Field4
	fieldBuf := make([]byte, 32)
	pos := int64(32)
	offset := uint32(1) // DBF records start with delete flag

	for {
		// Read one byte to check for terminator
		termBuf := make([]byte, 1)
		bytesRead := File4Read(&dataFile.File, pos, termBuf, 1)
		if bytesRead != 1 {
			return ErrorRead
		}

		// Check for header terminator
		if termBuf[0] == 0x0D {
			break // End of field definitions
		}

		// Read the full 32-byte field definition
		bytesRead = File4Read(&dataFile.File, pos, fieldBuf, 32)
		if bytesRead != 32 {
			return ErrorRead
		}

		field := &Field4{
			Data: nil, // Will be set when DATA4 is created
		}

		// Parse field name (first 11 bytes, null-terminated)
		copy(field.Name[:], fieldBuf[0:11])

		// Parse field type
		fieldType := fieldBuf[11]
		field.Type = int16(fieldType)

		// Skip offset field (bytes 12-15) for now

		// Parse field length and decimals
		field.Length = uint16(fieldBuf[16])
		field.Dec = uint16(fieldBuf[17])

		// Set field offset
		field.Offset = offset
		offset += uint32(field.Length)

		// Handle special field types
		switch fieldType {
		case FieldTypeChar, FieldTypeNumeric, FieldTypeFloat, FieldTypeDate, FieldTypeLogical, FieldTypeInteger, FieldTypeCurrency:
			// Standard field types - no special handling needed
		case FieldTypeMemo:
			// Initialize memo field handling
			field.Memo = &F4Memo{
				Field:     field,
				Status:    0,
				IsChanged: false,
				Contents:  nil,
				Length:    0,
				MaxLength: 0,
			}
			// Note: Memo file will be opened on first access if needed
		default:
			// Unknown field type - treat as character for now
			field.Type = int16(FieldTypeChar)
		}

		fields = append(fields, field)
		pos += 32 // Move to next field definition
	}

	// Set up the field array
	if len(fields) == 0 {
		return ErrorData
	}

	dataFile.Fields = fields
	dataFile.NumFields = int16(len(fields))

	return ErrorNone
}

// initBlankRecord initializes the blank record template
func initBlankRecord(data *Data4) {
	recordLen := int(data.DataFile.RecordLen)

	// Fill with spaces (standard DBF padding)
	for i := 0; i < recordLen; i++ {
		data.RecordBlank[i] = ' '
	}

	// Set delete flag to not deleted
	data.RecordBlank[0] = ' ' // ' ' = not deleted, '*' = deleted

	// Initialize logical fields to false ('F')
	for _, field := range data.Fields {
		if field.Type == int16(FieldTypeLogical) {
			offset := int(field.Offset)
			if offset < recordLen {
				data.RecordBlank[offset] = 'F' // False
			}
		}
	}
}

// D4Close closes a database file and releases all associated resources.
// This mirrors the d4close function from the CodeBase library.
//
// The function performs cleanup operations including:
// - Closing any associated memo files
// - Closing the main database file
// - Removing the database from the CODE4 data file list
//
// Returns ErrorNone on success, ErrorMemory if data is nil.
func D4Close(data *Data4) int {
	if data == nil {
		return ErrorMemory
	}

	// Close memo file if open
	if data.DataFile != nil && data.DataFile.MemoFile != nil {
		File4Close(&data.DataFile.MemoFile.File)
		data.DataFile.MemoFile = nil
	}

	// Close database file
	if data.DataFile != nil {
		File4Close(&data.DataFile.File)
	}

	// Remove from codebase data file list
	if data.CodeBase != nil {
		list4Remove(&data.CodeBase.DataFileList, &data.Link)
	}

	return ErrorNone
}

// D4Alias returns the alias name for the database.
// This mirrors the d4alias function from the CodeBase library.
//
// The alias is typically the base filename without path or extension,
// used for referencing the database in operations.
//
// Returns the alias string, empty string if data is nil.
func D4Alias(data *Data4) string {
	if data == nil {
		return ""
	}
	return data.Alias
}

// D4AliasSet sets the alias name for the database.
// This mirrors the d4aliasSet function from the CodeBase library.
//
// The alias is used to reference this database in CODE4 operations
// and should be unique within the CODE4 context.
//
// Parameters:
//   - data: Data4 structure to modify
//   - alias: New alias name to assign
func D4AliasSet(data *Data4, alias string) {
	if data != nil {
		data.Alias = alias
	}
}

// D4FileName returns the full path and filename of the database file.
// This mirrors the d4fileName function from the CodeBase library.
//
// Returns the complete file path as used when opening the database,
// empty string if data is nil or not properly initialized.
func D4FileName(data *Data4) string {
	if data == nil || data.DataFile == nil {
		return ""
	}
	return data.DataFile.File.Name
}

// D4NumFields returns the number of fields in the database.
// This mirrors the d4numFields function from the CodeBase library.
//
// Returns the count of fields defined in the database structure,
// 0 if data is nil or not properly initialized.
func D4NumFields(data *Data4) int16 {
	if data == nil || data.DataFile == nil {
		return 0
	}
	return data.DataFile.NumFields
}

// D4RecWidth returns the width (size in bytes) of each record.
// This mirrors the d4recWidth function from the CodeBase library.
//
// The record width includes all field lengths plus the delete flag byte.
//
// Returns the record width in bytes, 0 if data is nil.
func D4RecWidth(data *Data4) uint16 {
	if data == nil || data.DataFile == nil {
		return 0
	}
	return data.DataFile.RecordLen
}

// D4RecCount returns the total number of records in the database.
// This mirrors the d4recCount function from the CodeBase library.
//
// The count includes all records (both active and deleted) as stored
// in the database file header.
//
// Returns the total record count, 0 if data is nil.
func D4RecCount(data *Data4) int32 {
	if data == nil || data.DataFile == nil {
		return 0
	}
	return data.DataFile.Header.NumRecs
}

// D4RecNo returns the current record number (1-based).
// This mirrors the d4recNo function from the CodeBase library.
//
// Returns the record number of the currently positioned record,
// 0 if at BOF or data is nil.
func D4RecNo(data *Data4) int32 {
	if data == nil {
		return 0
	}
	return data.recNo
}

// D4Eof returns true if positioned at end-of-file.
// This mirrors the d4eof function from the CodeBase library.
//
// Returns true if the record pointer is beyond the last record,
// true if data is nil (safe default).
func D4Eof(data *Data4) bool {
	if data == nil {
		return true
	}
	return data.atEof
}

// D4Bof returns true if positioned at beginning-of-file.
// This mirrors the d4bof function from the CodeBase library.
//
// Returns true if the record pointer is before the first record,
// true if data is nil (safe default).
func D4Bof(data *Data4) bool {
	if data == nil {
		return true
	}
	return data.atBof
}

// D4Record returns the current record buffer.
// This mirrors the d4record function from the CodeBase library.
//
// The returned slice contains the raw record data including the
// delete flag and all field values as stored in the file.
//
// Returns the record buffer slice, nil if data is nil.
func D4Record(data *Data4) []byte {
	if data == nil {
		return nil
	}
	return data.Record
}

// D4RecordOld returns the previous record buffer.
// This mirrors the d4recordOld function from the CodeBase library.
//
// The returned slice contains the record data from before the last
// record movement operation, useful for undo operations.
//
// Returns the old record buffer slice, nil if data is nil.
func D4RecordOld(data *Data4) []byte {
	if data == nil {
		return nil
	}
	return data.RecordOld
}

// D4Blank initializes the current record with blank/default values.
// This mirrors the d4blank function from the CodeBase library.
//
// The function fills the current record buffer with the template
// blank record containing appropriate default values for each field type.
func D4Blank(data *Data4) {
	if data == nil || data.RecordBlank == nil {
		return
	}
	copy(data.Record, data.RecordBlank)
}

// D4Go positions the database to a specific record number.
// This mirrors the d4go function from the CodeBase library.
//
// The function reads the specified record into the current record buffer
// and updates the position state. Record numbers are 1-based.
//
// Parameters:
//   - data: Data4 structure representing the database
//   - recordNum: Record number to position to (1-based)
//
// Returns ErrorNone on success, ErrorMemory if data is nil,
// ErrorData if recordNum is out of range, ErrorRead on I/O errors.
func D4Go(data *Data4, recordNum int32) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	if recordNum < 1 || recordNum > data.DataFile.Header.NumRecs {
		return ErrorData
	}

	// Calculate file position
	headerLen := int64(data.DataFile.Header.HeaderLen)
	recordLen := int64(data.DataFile.RecordLen)
	pos := headerLen + (int64(recordNum-1) * recordLen)

	// Save old record
	if data.Record != nil {
		copy(data.RecordOld, data.Record)
	}

	// Read record
	bytesRead := File4Read(&data.DataFile.File, pos, data.Record, uint32(recordLen))
	if bytesRead != uint32(recordLen) {
		return ErrorRead
	}

	// Update position state
	data.recNo = recordNum
	data.atEof = false // We're on a valid record, not at EOF
	data.atBof = (recordNum == 1)

	return ErrorNone
}

// D4Top positions the database to the first record.
// This mirrors the d4top function from the CodeBase library.
//
// If the database is empty, the function sets EOF and BOF flags
// appropriately without generating an error.
//
// Returns ErrorNone on success, ErrorMemory if data is nil.
func D4Top(data *Data4) int {
	if data == nil {
		return ErrorMemory
	}

	if data.DataFile.Header.NumRecs == 0 {
		data.atEof = true
		data.atBof = true
		data.recNo = 0
		return ErrorNone
	}

	return D4Go(data, 1)
}

// D4Bottom positions the database to the last record.
// This mirrors the d4bottom function from the CodeBase library.
//
// If the database is empty, the function sets EOF and BOF flags
// appropriately without generating an error.
//
// Returns ErrorNone on success, ErrorMemory if data is nil.
func D4Bottom(data *Data4) int {
	if data == nil {
		return ErrorMemory
	}

	if data.DataFile.Header.NumRecs == 0 {
		data.atEof = true
		data.atBof = true
		data.recNo = 0
		return ErrorNone
	}

	return D4Go(data, data.DataFile.Header.NumRecs)
}

// D4Skip moves the record pointer by a relative number of records.
// This mirrors the d4skip function from the CodeBase library.
//
// Positive values move forward, negative values move backward.
// If the movement would go beyond the file boundaries, the appropriate
// EOF or BOF condition is set without generating an error.
//
// Parameters:
//   - data: Data4 structure representing the database
//   - numRecs: Number of records to skip (positive=forward, negative=backward)
//
// Returns ErrorNone on success, ErrorMemory if data is nil.
func D4Skip(data *Data4, numRecs int32) int {
	if data == nil {
		return ErrorMemory
	}

	newRecNo := data.recNo + numRecs

	// Handle boundary conditions
	if newRecNo < 1 {
		data.atBof = true
		data.atEof = false
		data.recNo = 0
		return ErrorNone
	}

	if newRecNo > data.DataFile.Header.NumRecs {
		data.atEof = true
		data.atBof = false
		data.recNo = data.DataFile.Header.NumRecs + 1
		return ErrorNone
	}

	return D4Go(data, newRecNo)
}

// Helper function to get field name as string
func getFieldName(field *Field4) string {
	nameBytes := field.Name[:]
	nullIndex := bytes.IndexByte(nameBytes, 0)
	if nullIndex >= 0 {
		return string(nameBytes[:nullIndex])
	}
	return string(nameBytes[:])
}

// D4Field finds and returns a field by its name.
// This mirrors the d4field function from the CodeBase library.
//
// The search is case-insensitive and returns the first matching field.
// The returned Field4 structure has its Data back-reference set for
// convenient field value access.
//
// Parameters:
//   - data: Data4 structure representing the database
//   - fieldName: Name of the field to find (case-insensitive)
//
// Returns a pointer to the Field4 structure if found, nil otherwise.
func D4Field(data *Data4, fieldName string) *Field4 {
	if data == nil || fieldName == "" {
		return nil
	}

	upperName := strings.ToUpper(fieldName)

	for _, field := range data.Fields {
		fieldNameStr := strings.ToUpper(getFieldName(field))
		if fieldNameStr == upperName {
			field.Data = data // Set back-reference
			return field
		}
	}

	return nil
}

// D4FieldJ returns a field by its position index.
// This mirrors the d4fieldJ function from the CodeBase library.
//
// Field indexes are 1-based (first field is index 1).
// The returned Field4 structure has its Data back-reference set.
//
// Parameters:
//   - data: Data4 structure representing the database
//   - fieldIndex: 1-based index of the field to retrieve
//
// Returns a pointer to the Field4 structure if found, nil if out of range.
func D4FieldJ(data *Data4, fieldIndex int) *Field4 {
	if data == nil || fieldIndex < 1 || fieldIndex > len(data.Fields) {
		return nil
	}

	field := data.Fields[fieldIndex-1] // Convert to 0-based
	field.Data = data                  // Set back-reference
	return field
}

// D4FieldNumber returns the 1-based index of a field by name.
// This mirrors the d4fieldNumber function from the CodeBase library.
//
// The search is case-insensitive and returns the position of the
// first matching field in the database structure.
//
// Parameters:
//   - data: Data4 structure representing the database
//   - fieldName: Name of the field to find (case-insensitive)
//
// Returns the 1-based field index, 0 if not found.
func D4FieldNumber(data *Data4, fieldName string) int {
	if data == nil || fieldName == "" {
		return 0
	}

	upperName := strings.ToUpper(fieldName)

	for i, field := range data.Fields {
		fieldNameStr := strings.ToUpper(getFieldName(field))
		if fieldNameStr == upperName {
			return i + 1 // Convert to 1-based
		}
	}

	return 0 // Field not found
}

// extractAlias extracts the alias (base filename without extension) from a filename
func extractAlias(fileName string) string {
	base := filepath.Base(fileName)
	// Remove extension
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return base
}

// hasMemoFields checks if the database contains any memo fields
func hasMemoFields(dataFile *Data4File) bool {
	for _, field := range dataFile.Fields {
		if field.Type == int16(FieldTypeMemo) {
			return true
		}
	}
	return false
}

// openMemoFile opens the associated memo file for a database
func openMemoFile(dataFile *Data4File, dbfFileName string) int {
	// Construct memo file name by changing extension to .fpt
	memoFileName := constructPath(dbfFileName, "fpt")

	// Create memo file structure
	memoFile := &Memo4File{
		BlockSize: 64, // Default block size
		Data:      dataFile,
	}

	// Try to open the memo file
	err := File4Open(&memoFile.File, dataFile.CodeBase, memoFileName, AccessDenyNone)
	if err != ErrorNone {
		return err
	}

	// Read memo file header to get block size
	headerBuf := make([]byte, 8)
	bytesRead := File4Read(&memoFile.File, 0, headerBuf, 8)
	if bytesRead == 8 {
		// Block size is stored at offset 6-7 in big endian for FPT files
		memoFile.BlockSize = int16(binary.BigEndian.Uint16(headerBuf[6:8]))
		if memoFile.BlockSize == 0 {
			memoFile.BlockSize = 512 // FPT default block size
		}
	} else {
		memoFile.BlockSize = 512 // Default FPT block size
	}

	dataFile.MemoFile = memoFile
	return ErrorNone
}
