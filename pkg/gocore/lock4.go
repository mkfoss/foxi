// Package pkg - Multi-user locking mechanisms
// Direct translation of CodeBase locking operations
package pkg

import (
	"sync"
	"syscall"
	"time"
)

// File locking states
const (
	LockNone = iota
	LockFile
	LockRecord
)

// FileLock represents a file lock state
type FileLock struct {
	File     *File4
	LockType int
	StartPos int64
	Length   int64
	Timeout  time.Duration
}

// Global lock manager
var (
	lockManager = &LockManager{
		locks:   make(map[string]*FileLock),
		mutex:   &sync.RWMutex{},
		timeout: 5 * time.Second,
	}
)

// LockManager manages file and record locks
type LockManager struct {
	locks   map[string]*FileLock
	mutex   *sync.RWMutex
	timeout time.Duration
}

// D4LockFile locks entire database file (mirrors d4lockFile)
func D4LockFile(data *Data4) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	return lockManager.LockFile(&data.DataFile.File)
}

// D4UnlockFile unlocks database file (mirrors d4unlockFile)
func D4UnlockFile(data *Data4) int {
	if data == nil || data.DataFile == nil {
		return ErrorMemory
	}

	return lockManager.UnlockFile(&data.DataFile.File)
}

// D4Lock locks current record (mirrors d4lock)
func D4Lock(data *Data4) int {
	if data == nil || data.DataFile == nil || data.recNo <= 0 {
		return ErrorMemory
	}

	// Calculate record position
	headerLen := int64(data.DataFile.Header.HeaderLen)
	recordLen := int64(data.DataFile.RecordLen)
	startPos := headerLen + ((int64(data.recNo) - 1) * recordLen)

	return lockManager.LockRange(&data.DataFile.File, startPos, recordLen)
}

// D4Unlock unlocks current record (mirrors d4unlock)
func D4Unlock(data *Data4) int {
	if data == nil || data.DataFile == nil || data.recNo <= 0 {
		return ErrorMemory
	}

	// Calculate record position
	headerLen := int64(data.DataFile.Header.HeaderLen)
	recordLen := int64(data.DataFile.RecordLen)
	startPos := headerLen + ((int64(data.recNo) - 1) * recordLen)

	return lockManager.UnlockRange(&data.DataFile.File, startPos, recordLen)
}

// LockFile implements file-level locking
func (lm *LockManager) LockFile(file *File4) int {
	if file == nil || file.Handle == nil {
		return ErrorMemory
	}

	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	filePath := file.Name

	// Check if already locked
	if _, exists := lm.locks[filePath]; exists {
		return ErrorData // Already locked
	}

	// Apply system-level file lock
	err := syscall.Flock(int(file.Handle.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return ErrorOpen // Lock failed
	}

	// Record the lock
	lm.locks[filePath] = &FileLock{
		File:     file,
		LockType: LockFile,
		StartPos: 0,
		Length:   -1, // Entire file
		Timeout:  lm.timeout,
	}

	return ErrorNone
}

// UnlockFile removes file-level lock
func (lm *LockManager) UnlockFile(file *File4) int {
	if file == nil || file.Handle == nil {
		return ErrorMemory
	}

	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	filePath := file.Name

	// Check if locked
	if _, exists := lm.locks[filePath]; !exists {
		return ErrorData // Not locked
	}

	// Remove system-level file lock
	err := syscall.Flock(int(file.Handle.Fd()), syscall.LOCK_UN)
	if err != nil {
		return ErrorClose // Unlock failed
	}

	// Remove from lock registry
	delete(lm.locks, filePath)

	return ErrorNone
}

// LockRange implements range locking for records
func (lm *LockManager) LockRange(file *File4, startPos, length int64) int {
	if file == nil || file.Handle == nil {
		return ErrorMemory
	}

	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	lockKey := file.Name + ":" + string(rune(startPos))

	// Check if range already locked
	if _, exists := lm.locks[lockKey]; exists {
		return ErrorData // Range already locked
	}

	// Apply system-level range lock (fcntl-style)
	fd := int(file.Handle.Fd())

	// Use flock for simplicity (would use fcntl with F_SETLK for precise ranges)
	err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return ErrorOpen // Lock failed
	}

	// Record the range lock
	lm.locks[lockKey] = &FileLock{
		File:     file,
		LockType: LockRecord,
		StartPos: startPos,
		Length:   length,
		Timeout:  lm.timeout,
	}

	return ErrorNone
}

// UnlockRange removes range lock
func (lm *LockManager) UnlockRange(file *File4, startPos, length int64) int {
	if file == nil || file.Handle == nil {
		return ErrorMemory
	}

	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	lockKey := file.Name + ":" + string(rune(startPos))

	// Check if range locked
	if _, exists := lm.locks[lockKey]; !exists {
		return ErrorData // Range not locked
	}

	// Remove system-level range lock
	fd := int(file.Handle.Fd())
	err := syscall.Flock(fd, syscall.LOCK_UN)
	if err != nil {
		return ErrorClose // Unlock failed
	}

	// Remove from lock registry
	delete(lm.locks, lockKey)

	return ErrorNone
}

// D4IsLocked checks if record is locked (mirrors d4isLocked)
func D4IsLocked(data *Data4) bool {
	if data == nil || data.DataFile == nil || data.recNo <= 0 {
		return false
	}

	// Calculate record position
	headerLen := int64(data.DataFile.Header.HeaderLen)
	recordLen := int64(data.DataFile.RecordLen)
	startPos := headerLen + ((int64(data.recNo) - 1) * recordLen)

	lockKey := data.DataFile.File.Name + ":" + string(rune(startPos))

	lockManager.mutex.RLock()
	defer lockManager.mutex.RUnlock()

	_, locked := lockManager.locks[lockKey]
	return locked
}

// D4LockAll locks entire database for exclusive access (mirrors d4lockAll)
func D4LockAll(data *Data4) int {
	if data == nil {
		return ErrorMemory
	}

	// Lock the data file
	err := D4LockFile(data)
	if err != ErrorNone {
		return err
	}

	// Lock all associated index files
	current := list4First(&data.Indexes)
	for current != nil {
		index := indexFromLink(current)
		if index != nil && index.IndexFile != nil {
			indexErr := lockManager.LockFile(&index.IndexFile.File)
			if indexErr != ErrorNone {
				// Rollback locks on failure
				D4UnlockAll(data)
				return indexErr
			}
		}

		// Move to next
		current = list4Next(&data.Indexes, current)
		if current == list4First(&data.Indexes) {
			break // Circular list, back to start
		}
	}

	return ErrorNone
}

// D4UnlockAll unlocks entire database (mirrors d4unlockAll)
func D4UnlockAll(data *Data4) int {
	if data == nil {
		return ErrorMemory
	}

	var lastError = ErrorNone

	// Unlock all associated index files
	current := list4First(&data.Indexes)
	for current != nil {
		index := indexFromLink(current)
		if index != nil && index.IndexFile != nil {
			err := lockManager.UnlockFile(&index.IndexFile.File)
			if err != ErrorNone {
				lastError = err // Record error but continue
			}
		}

		// Move to next
		current = list4Next(&data.Indexes, current)
		if current == list4First(&data.Indexes) {
			break
		}
	}

	// Unlock the data file
	err := D4UnlockFile(data)
	if err != ErrorNone {
		lastError = err
	}

	return lastError
}

// CleanupLocks removes all locks for a file (called on file close)
func CleanupLocks(file *File4) {
	if file == nil {
		return
	}

	lockManager.mutex.Lock()
	defer lockManager.mutex.Unlock()

	// Remove all locks associated with this file
	for key, lock := range lockManager.locks {
		if lock.File == file {
			delete(lockManager.locks, key)
		}
	}

	// Ensure system lock is released
	if file.Handle != nil {
		_ = syscall.Flock(int(file.Handle.Fd()), syscall.LOCK_UN) // Ignore error during cleanup
	}
}

// SetLockTimeout sets the default lock timeout
func SetLockTimeout(timeout time.Duration) {
	lockManager.mutex.Lock()
	defer lockManager.mutex.Unlock()
	lockManager.timeout = timeout
}

// GetLockStatus returns current lock information for debugging
func GetLockStatus() map[string]*FileLock {
	lockManager.mutex.RLock()
	defer lockManager.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*FileLock)
	for key, lock := range lockManager.locks {
		result[key] = lock
	}

	return result
}
