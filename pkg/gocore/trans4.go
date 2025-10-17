// Package pkg - Transaction support functions
// Direct translation of CodeBase transaction operations
package pkg

import (
	"time"
)


// Code4TransInit initializes transaction system (mirrors code4transInit)
func Code4TransInit(cb *Code4) int {
	if cb == nil {
		return ErrorMemory
	}

	// Initialize transaction level
	cb.TransactionLevel = 1
	cb.TransactionID = time.Now().UnixNano()

	// Initialize transaction log
	if cb.TransactionLog == nil {
		cb.TransactionLog = make([]*Trans4State, 0, 100)
	}

	return ErrorNone
}

// Code4TransCommit commits current transaction (mirrors code4transCommit)
func Code4TransCommit(cb *Code4) int {
	if cb == nil || cb.TransactionLevel == 0 {
		return ErrorMemory
	}

	// Flush all pending writes to disk
	current := list4First(&cb.DataFileList)
	for current != nil {
		data := data4FromLink(current)
		if data != nil && data.TransChanged != 0 {
			// Write current record if positioned
			if data.recNo > 0 {
				D4Write(data)
			}
			// Flush the file
			File4Flush(&data.DataFile.File)
			// Clear changed flag
			data.TransChanged = 0
		}

		// Move to next
		current = list4Next(&cb.DataFileList, current)
		if current == list4First(&cb.DataFileList) {
			break // Circular list, back to start
		}
	}

	// Clear transaction log
	cb.TransactionLog = cb.TransactionLog[:0]
	cb.TransactionLevel = 0
	cb.TransactionID = 0

	return ErrorNone
}

// Code4TransRollback rolls back current transaction (mirrors code4transRollback)
func Code4TransRollback(cb *Code4) int {
	if cb == nil || cb.TransactionLevel == 0 {
		return ErrorMemory
	}

	// Rollback changes in reverse order
	for i := len(cb.TransactionLog) - 1; i >= 0; i-- {
		trans := cb.TransactionLog[i]
		if trans == nil {
			continue
		}

		// Find the data file for this transaction
		current := list4First(&cb.DataFileList)
		for current != nil {
			data := data4FromLink(current)
			if data != nil {
				// Rollback the specific operation
				switch trans.Operation {
				case 1: // Append - remove the record
					if data.DataFile.Header.NumRecs >= trans.RecNo {
						data.DataFile.Header.NumRecs--
						writeDbfHeader(data.DataFile)
					}

				case 2: // Update - restore old record
					if trans.OldRecord != nil && trans.RecNo > 0 {
						err := D4Go(data, trans.RecNo)
						if err == ErrorNone {
							copy(data.Record, trans.OldRecord)
							D4Write(data)
						}
					}

				case 3: // Delete - undelete record
					if trans.RecNo > 0 {
						err := D4Go(data, trans.RecNo)
						if err == ErrorNone {
							D4Recall(data)
							D4Write(data)
						}
					}
				}
				break
			}

			// Move to next
			current = list4Next(&cb.DataFileList, current)
			if current == list4First(&cb.DataFileList) {
				break
			}
		}
	}

	// Clear transaction log
	cb.TransactionLog = cb.TransactionLog[:0]
	cb.TransactionLevel = 0
	cb.TransactionID = 0

	return ErrorNone
}

// D4TransAppend logs an append operation for rollback
func D4TransAppend(data *Data4, recNo int32) {
	if data == nil || data.CodeBase == nil || data.CodeBase.TransactionLevel == 0 {
		return
	}

	trans := &Trans4State{
		RecNo:     recNo,
		Operation: 1, // Append
		TimeStamp: time.Now(),
	}

	data.CodeBase.TransactionLog = append(data.CodeBase.TransactionLog, trans)
}

// D4TransUpdate logs an update operation for rollback
func D4TransUpdate(data *Data4, recNo int32, oldRecord []byte) {
	if data == nil || data.CodeBase == nil || data.CodeBase.TransactionLevel == 0 {
		return
	}

	// Create a copy of the old record
	oldRecordCopy := make([]byte, len(oldRecord))
	copy(oldRecordCopy, oldRecord)

	trans := &Trans4State{
		RecNo:     recNo,
		OldRecord: oldRecordCopy,
		Operation: 2, // Update
		TimeStamp: time.Now(),
	}

	data.CodeBase.TransactionLog = append(data.CodeBase.TransactionLog, trans)
}

// D4TransDelete logs a delete operation for rollback
func D4TransDelete(data *Data4, recNo int32) {
	if data == nil || data.CodeBase == nil || data.CodeBase.TransactionLevel == 0 {
		return
	}

	trans := &Trans4State{
		RecNo:     recNo,
		Operation: 3, // Delete
		TimeStamp: time.Now(),
	}

	data.CodeBase.TransactionLog = append(data.CodeBase.TransactionLog, trans)
}

// Enhanced write operations with transaction support

// D4AppendTrans appends a record with transaction support
func D4AppendTrans(data *Data4) int {
	if data == nil {
		return ErrorMemory
	}

	// Perform the append
	err := D4Append(data)
	if err != ErrorNone {
		return err
	}

	// Log transaction for rollback capability
	D4TransAppend(data, data.recNo)

	return ErrorNone
}

// D4WriteTrans writes record with transaction support  
func D4WriteTrans(data *Data4) int {
	if data == nil || data.RecordOld == nil {
		return ErrorMemory
	}

	// Log the old record for rollback
	D4TransUpdate(data, data.recNo, data.RecordOld)

	// Perform the write
	err := D4Write(data)
	if err != ErrorNone {
		return err
	}

	// Update the old record buffer
	copy(data.RecordOld, data.Record)

	return ErrorNone
}

// D4DeleteTrans deletes record with transaction support
func D4DeleteTrans(data *Data4) int {
	if data == nil {
		return ErrorMemory
	}

	// Only log if not already deleted
	if !D4Deleted(data) {
		D4TransDelete(data, data.recNo)
	}

	// Perform the delete
	D4Delete(data)

	return ErrorNone
}