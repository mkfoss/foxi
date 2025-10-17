// Package pkg - Record writing and modification functions
// Direct translation of CodeBase record write operations
package pkg

import (
	"encoding/binary"
	"time"
)

// D4Append appends a new blank record to the database.
// This mirrors the d4append function from the CodeBase library.
//
// The function initializes a new record at the end of the database
// and positions the record pointer to it. The new record is blanked
// with default values appropriate for each field type.
//
// Returns ErrorNone on success, ErrorMemory if data is nil.
func D4Append(data *Data4) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	// First try to start the append operation
	err := D4AppendStart(data, 1)
	if err != ErrorNone {
		return err
	}

	// Blank the new record
	D4Blank(data)

	// The record is now ready for field assignments
	// The actual write happens when d4append() is called (or d4flush, d4close, etc.)
	
	return ErrorNone
}

// D4AppendStart prepares for appending records (mirrors d4appendStart)
func D4AppendStart(data *Data4, lockAttempt int) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	dataFile := data.DataFile

	// Calculate new record position
	newRecordNo := dataFile.Header.NumRecs + 1

	// Update header record count
	dataFile.Header.NumRecs = newRecordNo

	// Set current position to the new record
	data.recNo = newRecordNo
	data.atEof = false
	data.atBof = (newRecordNo == 1)

	// Mark as changed for transaction tracking
	data.TransChanged = 1
	
	// Initialize transaction state if not already done
	if data.CodeBase != nil && data.CodeBase.TransactionLevel == 0 {
		// Auto-start transaction for safety
		Code4TransInit(data.CodeBase)
	}

	return ErrorNone
}

// D4AppendBlank appends a blank record (mirrors d4appendBlank)
func D4AppendBlank(data *Data4) int {
	err := D4Append(data)
	if err != ErrorNone {
		return err
	}

	// Record is already blanked by D4Append
	return D4Flush(data)
}

// D4Write writes the current record buffer to disk.
// This mirrors the d4write function from the CodeBase library.
//
// The function writes the current record data to the appropriate
// position in the database file and updates the header if necessary.
//
// Returns ErrorNone on success, ErrorMemory if data is nil.
func D4Write(data *Data4) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	return d4WriteLow(data, data.recNo, 1)
}

// d4WriteLow internal function for writing records (mirrors d4writeLow)
func d4WriteLow(data *Data4, recordNo int32, doFlush int) int {
	if data == nil || data.DataFile == nil || data.Record == nil {
		return ErrorMemory
	}

	dataFile := data.DataFile

	// Calculate file position for the record
	headerLen := int64(dataFile.Header.HeaderLen)
	recordLen := int64(dataFile.RecordLen)
	pos := headerLen + (int64(recordNo-1) * recordLen)

	// Write the record data
	err := File4Write(&dataFile.File, pos, data.Record, uint32(recordLen))
	if err != ErrorNone {
		return err
	}

	// Update header if we're writing beyond current record count
	if recordNo > dataFile.Header.NumRecs {
		dataFile.Header.NumRecs = recordNo
		err = writeDbfHeader(dataFile)
		if err != ErrorNone {
			return err
		}
	}

	// Flush if requested
	if doFlush != 0 {
		err = File4Flush(&dataFile.File)
		if err != ErrorNone {
			return err
		}
	}

	return ErrorNone
}

// writeDbfHeader writes the DBF header to disk
func writeDbfHeader(dataFile *Data4File) int {
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

	// Copy reserved area
	copy(headerBuf[12:28], header.Reserved[:])

	// Write header to file
	err := File4Write(&dataFile.File, 0, headerBuf, 32)
	if err != ErrorNone {
		return err
	}

	return ErrorNone
}

// D4Delete marks the current record as deleted.
// This mirrors the d4delete function from the CodeBase library.
//
// The function sets the delete flag (first byte) of the current record
// to '*' indicating the record is deleted. The record remains in the
// file but will be skipped during normal navigation.
func D4Delete(data *Data4) {
	if data == nil || data.Record == nil {
		return
	}

	// Set delete flag (first byte of record)
	data.Record[0] = '*' // '*' means deleted, ' ' means not deleted
}

// D4Deleted checks if current record is deleted (mirrors d4deleted)
func D4Deleted(data *Data4) bool {
	if data == nil || data.Record == nil {
		return false
	}

	// Check delete flag (first byte of record)
	return data.Record[0] == '*'
}

// D4Recall unmarks the current record as deleted (undeletes it).
// This mirrors the d4recall function from the CodeBase library.
//
// The function clears the delete flag (first byte) of the current record
// by setting it to ' ' (space), making the record active again.
func D4Recall(data *Data4) {
	if data == nil || data.Record == nil {
		return
	}

	// Clear delete flag (first byte of record)
	data.Record[0] = ' ' // ' ' means not deleted
}

// D4Flush flushes all pending writes to disk.
// This mirrors the d4flush function from the CodeBase library.
//
// The function ensures all record modifications are written to disk
// and forces the operating system to flush file buffers.
//
// Returns ErrorNone on success, ErrorMemory if data is nil.
func D4Flush(data *Data4) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	// Write current record if positioned
	if data.recNo > 0 && data.recNo <= data.DataFile.Header.NumRecs {
		err := D4Write(data)
		if err != ErrorNone {
			return err
		}
	}

	// Flush the file
	err := File4Flush(&data.DataFile.File)
	if err != ErrorNone {
		return err
	}

	// Flush memo file if present
	if data.DataFile.MemoFile != nil {
		err = File4Flush(&data.DataFile.MemoFile.File)
		if err != ErrorNone {
			return err
		}
	}

	return ErrorNone
}

// D4Update updates header with current date (mirrors d4update)  
func D4Update(data *Data4) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	// Update header date to current date
	now := time.Now()
	header := &data.DataFile.Header
	
	// DBF stores dates as YY/MM/DD since 1900
	year := now.Year()
	if year >= 2000 {
		header.Year = byte(year - 2000) // Y2K handling
	} else if year >= 1900 {
		header.Year = byte(year - 1900)
	}
	header.Month = byte(now.Month())
	header.Day = byte(now.Day())

	// Write updated header
	return writeDbfHeader(data.DataFile)
}

// D4Pack physically removes deleted records (mirrors d4pack)
func D4Pack(data *Data4) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	dataFile := data.DataFile
	recordLen := int(dataFile.RecordLen)
	originalCount := dataFile.Header.NumRecs
	newCount := int32(0)

	// Create temporary record buffer
	tempRecord := make([]byte, recordLen)

	// Scan through all records
	for i := int32(1); i <= originalCount; i++ {
		// Read record
		err := D4Go(data, i)
		if err != ErrorNone {
			return err
		}

		// If not deleted, write to packed position
		if !D4Deleted(data) {
			newCount++
			if newCount != i {
				// Copy record to new position
				copy(tempRecord, data.Record)
				err = d4WriteLow(data, newCount, 0)
				if err != ErrorNone {
					return err
				}
			}
		}
	}

	// Update record count in header
	dataFile.Header.NumRecs = newCount

	// Truncate file to new size
	headerLen := int64(dataFile.Header.HeaderLen)
	newFileSize := headerLen + (int64(newCount) * int64(recordLen))
	File4Truncate(&dataFile.File, newFileSize)

	// Write updated header
	err := writeDbfHeader(dataFile)
	if err != ErrorNone {
		return err
	}

	// Flush all changes
	err = File4Flush(&dataFile.File)
	if err != ErrorNone {
		return err
	}

	// Position to top
	if newCount > 0 {
		D4Top(data)
	} else {
		data.recNo = 0
		data.atEof = true
		data.atBof = true
	}

	return ErrorNone
}

// D4Zap removes all records from database (mirrors d4zap)
func D4Zap(data *Data4, startRec int32, numRecs int32) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	// If numRecs is 0 or -1, zap all records
	if numRecs <= 0 {
		startRec = 1
		numRecs = data.DataFile.Header.NumRecs
	}

	// Validate range
	if startRec < 1 || startRec > data.DataFile.Header.NumRecs {
		return ErrorData
	}

	endRec := startRec + numRecs - 1
	if endRec > data.DataFile.Header.NumRecs {
		endRec = data.DataFile.Header.NumRecs
	}

	// Mark records as deleted
	for i := startRec; i <= endRec; i++ {
		err := D4Go(data, i)
		if err != ErrorNone {
			return err
		}
		D4Delete(data)
		err = D4Write(data)
		if err != ErrorNone {
			return err
		}
	}

	// If zapping all records, update header count to 0
	if startRec == 1 && endRec == data.DataFile.Header.NumRecs {
		data.DataFile.Header.NumRecs = 0
		err := writeDbfHeader(data.DataFile)
		if err != ErrorNone {
			return err
		}
		
		// Position to empty state
		data.recNo = 0
		data.atEof = true
		data.atBof = true
	}

	return D4Flush(data)
}

// D4Replace replaces current record with data from another source (mirrors d4replace)
func D4Replace(data *Data4, sourceRecord []byte) int {
	if data == nil || data.Record == nil || sourceRecord == nil {
		return ErrorMemory
	}

	recordLen := len(data.Record)
	if len(sourceRecord) < recordLen {
		return ErrorData
	}

	// Copy source record to current record
	copy(data.Record, sourceRecord[:recordLen])

	return D4Write(data)
}

// D4Position returns relative position in database as percentage (mirrors d4position)
func D4Position(data *Data4) float64 {
	if data == nil || data.DataFile == nil {
		return 0.0
	}

	if data.DataFile.Header.NumRecs == 0 {
		return 0.0
	}

	if data.atEof {
		return 1.0
	}

	if data.atBof || data.recNo <= 0 {
		return 0.0
	}

	return float64(data.recNo) / float64(data.DataFile.Header.NumRecs)
}

// D4PositionSet moves to relative position in database (mirrors d4positionSet)
func D4PositionSet(data *Data4, position float64) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	if position < 0.0 {
		position = 0.0
	} else if position > 1.0 {
		position = 1.0
	}

	totalRecords := data.DataFile.Header.NumRecs
	if totalRecords == 0 {
		data.atEof = true
		data.atBof = true
		data.recNo = 0
		return ErrorNone
	}

	// Calculate target record number
	targetRec := int32(float64(totalRecords)*position + 0.5)
	if targetRec < 1 {
		targetRec = 1
	} else if targetRec > totalRecords {
		targetRec = totalRecords
	}

	return D4Go(data, targetRec)
}

// D4RefreshRecord refreshes current record from disk (mirrors d4refresh)
func D4RefreshRecord(data *Data4) int {
	if data == nil || data.recNo <= 0 {
		return ErrorMemory
	}

	// Re-read the current record from disk
	return D4Go(data, data.recNo)
}