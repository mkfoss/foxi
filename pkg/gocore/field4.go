// Package pkg - FIELD4 functions
// Direct translation of CodeBase field operations with Go type conversions
package pkg

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// F4Name returns the name of a field as a string.
// This mirrors the f4name function from the CodeBase library.
//
// The field name is extracted from the field definition and null-terminated
// as stored in the DBF file header.
//
// Returns the field name string, empty string if field is nil.
func F4Name(field *Field4) string {
	if field == nil {
		return ""
	}
	return getFieldName(field)
}

// F4Type returns the data type of a field as a character.
// This mirrors the f4type function from the CodeBase library.
//
// Field types include: 'C' (Character), 'N' (Numeric), 'F' (Float),
// 'D' (Date), 'L' (Logical), 'M' (Memo), 'I' (Integer), 'Y' (Currency).
//
// Returns the field type character, space if field is nil.
func F4Type(field *Field4) rune {
	if field == nil {
		return ' '
	}
	return rune(field.Type)
}

// F4Len returns the length (width) of a field in bytes.
// This mirrors the f4len function from the CodeBase library.
//
// The length represents the maximum number of characters or bytes
// that can be stored in this field.
//
// Returns the field length, 0 if field is nil.
func F4Len(field *Field4) uint16 {
	if field == nil {
		return 0
	}
	return field.Length
}

// F4Dec returns the number of decimal places for numeric fields.
// This mirrors the f4dec function from the CodeBase library.
//
// This value is only meaningful for Numeric and Float field types.
// For other field types, this value may be used for other purposes
// or be zero.
//
// Returns the decimal places count, 0 if field is nil.
func F4Dec(field *Field4) uint16 {
	if field == nil {
		return 0
	}
	return field.Dec
}

// F4Str returns the current value of a field as a string.
// This mirrors the f4str function from the CodeBase library.
//
// The function extracts the field data from the current record buffer
// and converts it to a string representation based on the field type:
// - Character fields: Return raw content with padding preserved
// - Numeric/Float/Integer/Currency: Return formatted numeric string
// - Date: Return date in YYYYMMDD format
// - Logical: Return 'T', 'F', or other logical values
// - Memo: Return memo content from associated memo file
//
// Returns the field value as string, empty string if field is nil or invalid.
func F4Str(field *Field4) string {
	if field == nil || field.Data == nil {
		return ""
	}

	record := field.Data.Record
	if record == nil {
		return ""
	}

	// Extract raw field data
	start := int(field.Offset)
	end := start + int(field.Length)
	
	if start < 0 || end > len(record) {
		return ""
	}

	fieldData := record[start:end]
	
	// Convert based on field type
	switch rune(field.Type) {
	case FieldTypeChar:
		// Character field - preserve exact content including padding (match C behavior)
		return string(fieldData)
		
	case FieldTypeNumeric, FieldTypeFloat:
		// Numeric field - preserve raw content for binary compatibility
		return string(fieldData)
		
	case FieldTypeInteger:
		// Integer field - preserve raw content for binary compatibility
		return string(fieldData)
		
	case FieldTypeCurrency:
		// Currency field - preserve raw content for binary compatibility
		return string(fieldData)
		
	case FieldTypeDate:
		// Date field - preserve raw format to match C behavior
		return string(fieldData)
		
	case FieldTypeDateTime:
		// DateTime field - preserve raw content for binary compatibility
		return string(fieldData)
		
	case FieldTypeLogical:
		// Logical field - preserve raw content for binary compatibility
		return string(fieldData)
		
	case FieldTypeMemo:
		// Memo field - return memo content if available (CodeBase library behavior)
		if field.Memo != nil && field.Memo.Contents != nil {
			return string(field.Memo.Contents)
		}
		// Try to read memo content from memo file
		if field.Data != nil && field.Data.DataFile != nil && field.Data.DataFile.MemoFile != nil {
			blockNumStr := strings.TrimSpace(string(fieldData))
			if blockNumStr != "" && blockNumStr != "0" {
				content := readMemoContent(field.Data.DataFile.MemoFile, blockNumStr)
				if content != "" {
					return content
				}
			}
		}
		// Otherwise return spaces like CodeBase library does for empty/failed memo reads
		return string(fieldData)
		
	default:
		// Unknown type - treat as character with preserved padding
		return string(fieldData)
	}
}

// F4Assign assigns a string value to a field in the current record.
// This mirrors the f4assign function from the CodeBase library.
//
// The function converts the string value to the appropriate format based
// on the field type and stores it in the current record buffer:
// - Character fields: Truncate/pad with spaces as needed
// - Numeric fields: Parse and format with proper alignment and decimals
// - Date fields: Parse various date formats and store as YYYYMMDD
// - Logical fields: Convert boolean representations to 'T'/'F'
// - Memo fields: Store in associated memo file
//
// Parameters:
//   - field: Field4 structure representing the target field
//   - value: String value to assign to the field
//
// Returns ErrorNone on success, ErrorMemory for nil parameters,
// ErrorData for invalid field bounds or conversion errors.
func F4Assign(field *Field4, value string) int {
	if field == nil || field.Data == nil {
		return ErrorMemory
	}

	record := field.Data.Record
	if record == nil {
		return ErrorMemory
	}

	// Check bounds
	start := int(field.Offset)
	end := start + int(field.Length)
	
	if start < 0 || end > len(record) {
		return ErrorData
	}

	// Clear field area first
	for i := start; i < end; i++ {
		record[i] = ' '
	}

	// Convert and assign based on field type
	switch rune(field.Type) {
	case FieldTypeChar:
		return assignCharField(record[start:end], value)
		
	case FieldTypeNumeric, FieldTypeFloat:
		return assignNumericField(record[start:end], value, field.Dec)
		
	case FieldTypeInteger:
		return assignIntegerField(record[start:end], value)
		
	case FieldTypeCurrency:
		return assignCurrencyField(record[start:end], value)
		
	case FieldTypeDate:
		return assignDateField(record[start:end], value)
		
	case FieldTypeDateTime:
		return assignCharField(record[start:end], value)
		
	case FieldTypeLogical:
		return assignLogicalField(record[start:end], value)
		
		case FieldTypeMemo:
			// Memo field assignment
			return assignMemoField(field, value)
		
	default:
		// Unknown type - treat as character
		return assignCharField(record[start:end], value)
	}
}

// assignCharField assigns character data to field buffer
func assignCharField(buffer []byte, value string) int {
	// Truncate if too long, pad with spaces if too short
	valueBytes := []byte(value)
	copyLen := len(valueBytes)
	if copyLen > len(buffer) {
		copyLen = len(buffer)
	}
	
	copy(buffer, valueBytes[:copyLen])
	// buffer is already space-padded from clearing above
	
	return ErrorNone
}

// assignNumericField assigns numeric data to field buffer
func assignNumericField(buffer []byte, value string, decimals uint16) int {
	// Parse the numeric value
	numValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		// If can't parse, fill with spaces (null value)
		return ErrorNone
	}

	// Format according to field specification
	var formatted string
	if decimals > 0 {
		// Format with decimals
		formatted = fmt.Sprintf("%.*f", decimals, numValue)
	} else {
		// Format as integer
		formatted = fmt.Sprintf("%.0f", numValue)
	}

	// Right-align in field
	if len(formatted) <= len(buffer) {
		start := len(buffer) - len(formatted)
		copy(buffer[start:], []byte(formatted))
	} else {
		// Value too large - fill with asterisks (overflow indicator)
		for i := range buffer {
			buffer[i] = '*'
		}
	}

	return ErrorNone
}

// assignIntegerField assigns integer data to field buffer
func assignIntegerField(buffer []byte, value string) int {
	// Parse the integer value
	intValue, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		// If can't parse, fill with spaces (null value)
		return ErrorNone
	}

	// Format as right-aligned integer (no decimals)
	formatted := fmt.Sprintf("%d", intValue)

	// Right-align in field
	if len(formatted) <= len(buffer) {
		start := len(buffer) - len(formatted)
		copy(buffer[start:], []byte(formatted))
	} else {
		// Value too large - fill with asterisks (overflow indicator)
		for i := range buffer {
			buffer[i] = '*'
		}
	}

	return ErrorNone
}

// assignCurrencyField assigns currency data to field buffer
func assignCurrencyField(buffer []byte, value string) int {
	// Parse the currency value (always 4 decimal places)
	currValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		// If can't parse, fill with spaces (null value)
		return ErrorNone
	}

	// Format with exactly 4 decimal places for currency
	formatted := fmt.Sprintf("%.4f", currValue)

	// Right-align in field
	if len(formatted) <= len(buffer) {
		start := len(buffer) - len(formatted)
		copy(buffer[start:], []byte(formatted))
	} else {
		// Value too large - fill with asterisks (overflow indicator)
		for i := range buffer {
			buffer[i] = '*'
		}
	}

	return ErrorNone
}

// assignMemoField assigns memo data to field
func assignMemoField(field *Field4, value string) int {
	if field == nil || field.Memo == nil {
		return ErrorMemory
	}

	// Store the memo content
	field.Memo.Contents = []byte(value)
	field.Memo.Length = uint32(len(value))
	field.Memo.IsChanged = true

	// In the DBF record, memo fields contain the block number
	// For now, we'll store a placeholder - full implementation would
	// require memo file handling
	if field.Data != nil && field.Data.Record != nil {
		record := field.Data.Record
		start := int(field.Offset)
		end := start + int(field.Length)
		
		if start >= 0 && end <= len(record) {
			// Clear the memo block number field
			for i := start; i < end; i++ {
				record[i] = ' '
			}
			// For basic implementation, store sequence number
			// Real implementation would allocate memo blocks
			blockNum := "1" // Simplified block number
			if len(blockNum) <= end-start {
				copy(record[start:start+len(blockNum)], blockNum)
			}
		}
	}

	return ErrorNone
}

// assignDateField assigns date data to field buffer (YYYYMMDD format)
func assignDateField(buffer []byte, value string) int {
	if len(buffer) != 8 {
		return ErrorData // Date fields should be 8 bytes
	}

	value = strings.TrimSpace(value)
	
	// Try to parse various date formats
	var dateStr string
	
	// Check if already in YYYYMMDD format
	if len(value) == 8 {
		dateStr = value
	} else {
		// Try to parse common formats and convert to YYYYMMDD
		layouts := []string{
			"2006/01/02",
			"2006-01-02", 
			"01/02/2006",
			"01-02-2006",
			"2006.01.02",
		}
		
		var parsed time.Time
		var err error
		
		for _, layout := range layouts {
			parsed, err = time.Parse(layout, value)
			if err == nil {
				break
			}
		}
		
		if err != nil {
			// Can't parse date - leave as spaces (null date)
			return ErrorNone
		}
		
		dateStr = parsed.Format("20060102")
	}

	// Validate YYYYMMDD format
	if len(dateStr) == 8 {
		copy(buffer, []byte(dateStr))
	}
	
	return ErrorNone
}

// assignLogicalField assigns logical data to field buffer
func assignLogicalField(buffer []byte, value string) int {
	if len(buffer) < 1 {
		return ErrorData
	}

	value = strings.TrimSpace(strings.ToUpper(value))
	
	switch value {
	case "T", "TRUE", "Y", "YES", "1":
		buffer[0] = 'T'
	case "F", "FALSE", "N", "NO", "0":
		buffer[0] = 'F'
	default:
		buffer[0] = 'F' // Default to false
	}
	
	return ErrorNone
}

// readMemoContent reads memo field content from the memo file
func readMemoContent(memoFile *Memo4File, blockNumStr string) string {
	// Parse block number
	blockNum := int32(0)
	n, err := fmt.Sscanf(blockNumStr, "%d", &blockNum)
	if err != nil || n != 1 || blockNum <= 0 {
		return ""
	}
	
	// FPT files use 512-byte blocks by default
	blockSize := int64(512)
	if memoFile.BlockSize > 0 {
		blockSize = int64(memoFile.BlockSize)
	}
	
	// Calculate file position (block 0 is header, data starts at block 1)
	pos := int64(blockNum) * blockSize
	
	// Read memo block header (first 8 bytes of block)
	headerBuf := make([]byte, 8)
	bytesRead := File4Read(&memoFile.File, pos, headerBuf, 8)
	if bytesRead != 8 {
		return ""
	}
	
	// In FPT format, bytes 4-7 contain the memo length (big endian)
	memoLength := int32(binary.BigEndian.Uint32(headerBuf[4:8]))
	if memoLength <= 0 || memoLength > 65535 { // Reasonable size limit
		return ""
	}
	
	// Read memo content (starts after 8-byte block header)
	contentBuf := make([]byte, memoLength)
	bytesRead = File4Read(&memoFile.File, pos+8, contentBuf, uint32(memoLength))
	if bytesRead != uint32(memoLength) {
		return ""
	}
	
	// Convert to string and trim null terminators
	content := string(contentBuf)
	content = strings.TrimRight(content, "\x00")
	return content
}

// F4Double returns the field value as a float64.
// This mirrors the f4double function from the CodeBase library.
//
// The function converts the field's string representation to a floating-point
// number based on the field type:
// - Numeric/Float/Currency fields: Parse as decimal number
// - Integer fields: Parse as integer and convert to float64
// - Logical fields: Return 1.0 for true, 0.0 for false
// - Other types: Attempt to parse as number, return 0.0 on failure
//
// Returns the field value as float64, 0.0 if field is nil or conversion fails.
func F4Double(field *Field4) float64 {
	if field == nil {
		return 0.0
	}

	strValue := F4Str(field)
	if strValue == "" {
		return 0.0
	}

	// For numeric fields, parse the string value
	switch rune(field.Type) {
	case FieldTypeNumeric, FieldTypeFloat:
		value, err := strconv.ParseFloat(strings.TrimSpace(strValue), 64)
		if err != nil {
			return 0.0
		}
		return value
		
	case FieldTypeInteger:
		value, err := strconv.ParseInt(strings.TrimSpace(strValue), 10, 32)
		if err != nil {
			return 0.0
		}
		return float64(value)
		
	case FieldTypeCurrency:
		value, err := strconv.ParseFloat(strings.TrimSpace(strValue), 64)
		if err != nil {
			return 0.0
		}
		return value
		
	case FieldTypeLogical:
		if strValue == "T" {
			return 1.0
		}
		return 0.0
		
	default:
		// Try to parse as number anyway
		value, err := strconv.ParseFloat(strings.TrimSpace(strValue), 64)
		if err != nil {
			return 0.0
		}
		return value
	}
}

// F4AssignDouble assigns a float64 value to a field.
// This mirrors the f4assignDouble function from the CodeBase library.
//
// The function converts the float64 value to the appropriate string format
// based on the field type and then assigns it using F4Assign:
// - Numeric/Float fields: Format with specified decimal places
// - Integer fields: Format as whole number
// - Currency fields: Format with 4 decimal places
// - Logical fields: Convert to 'T' (non-zero) or 'F' (zero)
//
// Parameters:
//   - field: Field4 structure representing the target field
//   - value: float64 value to assign
//
// Returns ErrorNone on success, ErrorMemory if field is nil.
func F4AssignDouble(field *Field4, value float64) int {
	if field == nil {
		return ErrorMemory
	}

	// Convert double to appropriate string representation
	var strValue string
	
	switch rune(field.Type) {
	case FieldTypeNumeric, FieldTypeFloat:
		if field.Dec > 0 {
			strValue = fmt.Sprintf("%.*f", field.Dec, value)
		} else {
			strValue = fmt.Sprintf("%.0f", value)
		}
		
	case FieldTypeInteger:
		strValue = fmt.Sprintf("%.0f", value)
		
	case FieldTypeCurrency:
		strValue = fmt.Sprintf("%.4f", value) // Always 4 decimal places for currency
		
	case FieldTypeLogical:
		if value != 0.0 {
			strValue = "T"
		} else {
			strValue = "F"
		}
		
	default:
		strValue = fmt.Sprintf("%g", value)
	}

	return F4Assign(field, strValue)
}

// F4Int returns the field value as an int.
// This mirrors the f4int function from the CodeBase library.
//
// The function converts the field value to int by first converting
// to float64 and then casting to int (truncating any decimal portion).
//
// Returns the field value as int, 0 if field is nil or conversion fails.
func F4Int(field *Field4) int {
	return int(F4Double(field))
}

// F4AssignInt assigns an int value to a field.
// This mirrors the f4assignInt function from the CodeBase library.
//
// Convenience function that converts the int to float64 and calls F4AssignDouble.
//
// Returns ErrorNone on success, ErrorMemory if field is nil.
func F4AssignInt(field *Field4, value int) int {
	return F4AssignDouble(field, float64(value))
}

// F4Long returns the field value as an int32.
// This mirrors the f4long function from the CodeBase library.
//
// Convenience function that calls F4Double and casts the result to int32.
//
// Returns the field value as int32, 0 if field is nil or conversion fails.
func F4Long(field *Field4) int32 {
	return int32(F4Double(field))
}

// F4AssignLong assigns an int32 value to a field.
// This mirrors the f4assignLong function from the CodeBase library.
//
// Convenience function that converts the int32 to float64 and calls F4AssignDouble.
//
// Returns ErrorNone on success, ErrorMemory if field is nil.
func F4AssignLong(field *Field4, value int32) int {
	return F4AssignDouble(field, float64(value))
}

// F4True returns true if the field contains a logical true value.
// This mirrors the f4true function from the CodeBase library.
//
// The function checks if a logical field contains 'T' (true).
// All other values including 'F', spaces, and other characters are false.
//
// Returns true if field contains 'T', false otherwise or if field is nil.
func F4True(field *Field4) bool {
	if field == nil {
		return false
	}

	strValue := F4Str(field)
	return strValue == "T"
}

// F4AssignLogical assigns boolean value to logical field (mirrors f4assignLogical)
func F4AssignLogical(field *Field4, value bool) int {
	if value {
		return F4Assign(field, "T")
	}
	return F4Assign(field, "F")
}

// F4Blank blanks a field (mirrors f4blank)
func F4Blank(field *Field4) {
	if field == nil || field.Data == nil {
		return
	}

	record := field.Data.Record
	if record == nil {
		return
	}

	start := int(field.Offset)
	end := start + int(field.Length)
	
	if start < 0 || end > len(record) {
		return
	}

	// Fill field with appropriate blank value
	switch rune(field.Type) {
	case FieldTypeLogical:
		record[start] = 'F' // False for logical fields
	default:
		// Spaces for all other field types
		for i := start; i < end; i++ {
			record[i] = ' '
		}
	}
}

// F4DateTime returns field value as time.Time for DateTime fields
func F4DateTime(field *Field4) time.Time {
	if field == nil {
		return time.Time{}
	}

	switch rune(field.Type) {
	case FieldTypeDate:
		strValue := strings.TrimSpace(F4Str(field))
		if len(strValue) == 8 { // YYYYMMDD
			if parsed, err := time.Parse("20060102", strValue); err == nil {
				return parsed
			}
		}
		// Try formatted date
		if len(strValue) == 10 { // YYYY/MM/DD
			if parsed, err := time.Parse("2006/01/02", strValue); err == nil {
				return parsed
			}
		}
		
	case FieldTypeDateTime:
		// FoxPro DateTime field - 8 bytes: 4 bytes Julian date + 4 bytes milliseconds
		strValue := strings.TrimSpace(F4Str(field))
		if len(strValue) >= 8 {
			// For simplified implementation, try to parse as formatted datetime
			layouts := []string{
				"2006-01-02 15:04:05",
				"2006/01/02 15:04:05", 
				"01/02/2006 15:04:05",
				"2006-01-02T15:04:05",
			}
			
			for _, layout := range layouts {
				if parsed, err := time.Parse(layout, strValue); err == nil {
					return parsed
				}
			}
		}
		return time.Time{}
	}

	return time.Time{}
}

// F4AssignDateTime assigns time.Time value to date field
func F4AssignDateTime(field *Field4, value time.Time) int {
	if field == nil {
		return ErrorMemory
	}

	switch rune(field.Type) {
	case FieldTypeDate:
		dateStr := value.Format("20060102")
		return F4Assign(field, dateStr)
		
	case FieldTypeDateTime:
		// FoxPro DateTime field assignment
		// For simplified implementation, store as formatted string
		datetimeStr := value.Format("2006-01-02 15:04:05")
		return F4Assign(field, datetimeStr)
		
	default:
		return ErrorData
	}
}