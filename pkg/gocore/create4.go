// Package pkg - Database creation functions
// Direct translation of CodeBase database creation operations
package pkg

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// D4Create creates a new Visual FoxPro DBF database file with specified field structure.
// This mirrors the d4create function from the CodeBase library.
//
// The function performs the following operations:
// - Validates the field definitions and database parameters
// - Creates a new DBF file (removes existing if safety is off)
// - Writes the DBF header with current date and field count
// - Writes field descriptors for all specified fields
// - Initializes blank record templates with appropriate defaults
// - Sets up Data4 structure for immediate use
//
// Field types supported: Character ('C'), Numeric ('N'), Float ('F'),
// Date ('D'), Logical ('L'), Memo ('M'), Integer ('I'), Currency ('Y'), DateTime ('T')
//
// Parameters:
//   - cb: CODE4 context for database operations and settings
//   - fileName: Path for the new DBF file (.dbf extension added if missing)
//   - fieldInfo: Array of Field4Info structures defining the database schema
//
// Returns a pointer to the initialized Data4 structure ready for use,
// nil on failure (invalid parameters, file creation errors, etc.).
func D4Create(cb *Code4, fileName string, fieldInfo []Field4Info) *Data4 {
	if cb == nil || fileName == "" || len(fieldInfo) == 0 {
		return nil
	}

	// Construct full file path
	fullPath := fileName
	if filepath.Ext(fullPath) == "" {
		fullPath += ".dbf"
	}

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		if cb.Safety != 0 {
			return nil // File exists and safety is on
		}
		// Remove existing file
		os.Remove(fullPath)
	}

	// Create the data structures
	data := &Data4{
		CodeBase: cb,
		Alias:    strings.ToUpper(filepath.Base(strings.TrimSuffix(fullPath, filepath.Ext(fullPath)))),
	}

	dataFile := &Data4File{
		CodeBase: cb,
	}

	// Create and initialize the file
	err := File4Create(&dataFile.File, cb, fullPath, 1)
	if err != ErrorNone {
		return nil
	}

	// Calculate header and record layout
	numFields := int16(len(fieldInfo))
	headerLen := uint16(32 + (numFields * 32) + 1) // Base header + field descriptors + terminator
	recordLen := uint16(1)                         // Start with delete flag byte

	// Validate fields and calculate record length
	fields := make([]*Field4, 0, len(fieldInfo))
	for i, info := range fieldInfo {
		// Validate field info
		if info.Name == "" || len(info.Name) > 10 {
			File4Close(&dataFile.File)
			return nil
		}

		// Validate field type
		switch info.Type {
		case FieldTypeChar, FieldTypeNumeric, FieldTypeDate, FieldTypeLogical,
			FieldTypeMemo, FieldTypeFloat, FieldTypeInteger, FieldTypeCurrency, FieldTypeDateTime:
			// Valid types
		default:
			File4Close(&dataFile.File)
			return nil
		}

		// Create field structure
		field := &Field4{
			Type:   int16(info.Type),
			Length: info.Length,
			Dec:    info.Dec,
			Offset: uint32(recordLen),
			Data:   data,
		}

		// Set field name
		copy(field.Name[:], strings.ToUpper(info.Name))

		// Add field length to record length
		recordLen += info.Length

		fields = append(fields, field)

		// Set field number for array access
		if i == 0 {
			dataFile.Fields = make([]*Field4, len(fieldInfo))
		}
		dataFile.Fields[i] = field
	}

	// Initialize header
	now := time.Now()
	dataFile.Header = DbfHeader{
		Version:   0x03, // DBase III compatible
		Year:      byte(now.Year() - 1900),
		Month:     byte(now.Month()),
		Day:       byte(now.Day()),
		NumRecs:   0,
		HeaderLen: headerLen,
		RecordLen: recordLen,
	}

	dataFile.NumFields = numFields
	dataFile.RecordLen = recordLen

	// Write the header to file
	err = writeDbfHeaderComplete(dataFile)
	if err != ErrorNone {
		File4Close(&dataFile.File)
		return nil
	}

	// Write field descriptors
	err = writeFieldDescriptors(dataFile)
	if err != ErrorNone {
		File4Close(&dataFile.File)
		return nil
	}

	// Write header terminator
	terminator := []byte{0x0D}
	File4Write(&dataFile.File, int64(32+(numFields*32)), terminator, 1)

	// Set up data structure
	data.DataFile = dataFile
	data.Fields = fields
	data.RecordBlank = make([]byte, recordLen)
	data.Record = make([]byte, recordLen)
	data.RecordOld = make([]byte, recordLen)

	// Initialize blank record (all spaces except delete flag)
	data.RecordBlank[0] = ' ' // Not deleted
	for i := 1; i < int(recordLen); i++ {
		data.RecordBlank[i] = ' '
	}

	// Set logical fields to 'F' in blank record
	for _, field := range fields {
		if field.Type == int16(FieldTypeLogical) {
			data.RecordBlank[field.Offset] = 'F'
		}
	}

	copy(data.Record, data.RecordBlank)
	copy(data.RecordOld, data.RecordBlank)

	// Initialize navigation state
	data.recNo = 0
	data.atBof = true
	data.atEof = true

	// Add to CodeBase's data file list
	list4Add(&cb.DataFileList, &data.Link)

	dataFile.IsValid = true
	return data
}

// writeDbfHeaderComplete writes the complete DBF header to file
func writeDbfHeaderComplete(dataFile *Data4File) int {
	if dataFile == nil {
		return ErrorMemory
	}

	header := &dataFile.Header
	headerBuf := make([]byte, 32)

	// Pack header fields
	headerBuf[0] = header.Version
	headerBuf[1] = header.Year
	headerBuf[2] = header.Month
	headerBuf[3] = header.Day
	binary.LittleEndian.PutUint32(headerBuf[4:8], uint32(header.NumRecs))
	binary.LittleEndian.PutUint16(headerBuf[8:10], header.HeaderLen)
	binary.LittleEndian.PutUint16(headerBuf[10:12], header.RecordLen)

	// Reserved area (bytes 12-31)
	copy(headerBuf[12:32], header.Reserved[:])

	// Write header to file
	bytesWritten := File4Write(&dataFile.File, 0, headerBuf, 32)
	if bytesWritten != 32 {
		return ErrorWrite
	}

	return ErrorNone
}

// writeFieldDescriptors writes field descriptor array to file
func writeFieldDescriptors(dataFile *Data4File) int {
	if dataFile == nil || dataFile.Fields == nil {
		return ErrorMemory
	}

	for i, field := range dataFile.Fields {
		if field == nil {
			continue
		}

		// Create field descriptor (32 bytes)
		descriptor := make([]byte, 32)

		// Field name (11 bytes, null-terminated)
		fieldName := strings.TrimSpace(string(field.Name[:]))
		if len(fieldName) > 10 {
			fieldName = fieldName[:10]
		}
		copy(descriptor[0:11], fieldName)

		// Field type (1 byte)
		descriptor[11] = byte(field.Type)

		// Field offset (4 bytes) - not used in DBF format, set to 0
		binary.LittleEndian.PutUint32(descriptor[12:16], 0)

		// Field length (1 byte)
		descriptor[16] = byte(field.Length)

		// Decimal places (1 byte)
		descriptor[17] = byte(field.Dec)

		// Reserved bytes (14 bytes)
		for j := 18; j < 32; j++ {
			descriptor[j] = 0
		}

		// Write descriptor to file
		offset := int64(32 + (i * 32))
		bytesWritten := File4Write(&dataFile.File, offset, descriptor, 32)
		if bytesWritten != 32 {
			return ErrorWrite
		}
	}

	return ErrorNone
}

// D4CreateData creates a new database with extended options (mirrors d4createData)
func D4CreateData(cb *Code4, fileName string, fieldInfo []Field4Info, tag *Tag4) *Data4 {
	// Create the basic database first
	data := D4Create(cb, fileName, fieldInfo)
	if data == nil {
		return nil
	}

	// If tag is specified, create an associated index
	if tag != nil && tag.TagFile != nil {
		// Create associated CDX index
		tagInfo := []Tag4Info{
			{
				Name:       string(tag.TagFile.Alias[:]),
				Expression: tag.TagFile.ExprSource,
				Filter:     tag.TagFile.FilterSource,
				Unique:     0,
				Descending: 0,
			},
		}

		if tag.TagFile.Header.TypeCode&CDXTypeUnique != 0 {
			tagInfo[0].Unique = 1
		}
		if tag.TagFile.Header.Descending != 0 {
			tagInfo[0].Descending = 1
		}

		// Create the index
		index := I4Create(data, "", tagInfo)
		if index != nil {
			// Link the tag to the created index
			tag.Index = index
		}
	}

	return data
}
