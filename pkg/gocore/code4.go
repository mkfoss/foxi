// Package pkg - CODE4 functions
// Direct translation of CodeBase C API functions
package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// Code4Init initializes a CODE4 structure with default settings.
// This mirrors the code4init function from the CodeBase library.
//
// The function sets up default values for database access including:
// - Auto-open mode enabled
// - Safety checks enabled
// - Visual FoxPro 3.0 compatibility
// - CDX index extension as default
// - Windows ANSI code page (1252)
//
// Returns ErrorNone on success, ErrorMemory if cb is nil.
func Code4Init(cb *Code4) int {
	if cb == nil {
		return ErrorMemory
	}

	// Initialize with defaults matching C library
	cb.AutoOpen = true
	cb.CreateTemp = false
	cb.ErrDefaultUnique = 0 // r4unique equivalent
	cb.ErrExpr = 1
	cb.ErrFieldName = 1
	cb.ErrOff = 1
	cb.ErrOpen = 1
	cb.Log = 0
	cb.MemExpandData = 512
	cb.MemSizeBuffer = 8192
	cb.MemSizeSortBuffer = 8192
	cb.MemSizeSortPool = 8192
	cb.MemStartData = 2048
	cb.Safety = 1
	cb.Timeout = 0
	cb.Compatibility = 30 // VFP 3.0 compatibility

	// Internal initialization
	cb.Initialized = true
	cb.NumericStrLen = 10
	cb.Decimals = 2
	cb.ErrorCode = ErrorNone
	cb.FieldBuffer = make([]byte, 1024)
	copy(cb.IndexExtension[:], "CDX") // Default to CDX for FoxPro
	cb.CollatingSequence = 0
	cb.CodePage = 1252 // Windows ANSI

	// Initialize lists
	cb.DataFileList = List4{}

	return ErrorNone
}

// Code4InitUndo cleans up and releases resources from a CODE4 structure.
// This mirrors the code4initUndo function from the CodeBase library.
//
// The function closes all open data files and resets the CODE4 structure
// to an uninitialized state. It's safe to call multiple times.
//
// Returns ErrorNone on success.
func Code4InitUndo(cb *Code4) int {
	if cb == nil || !cb.Initialized {
		return ErrorNone
	}

	// Close all open data files
	Code4Close(cb)

	// Reset structure
	cb.Initialized = false
	cb.ErrorCode = ErrorNone
	cb.FieldBuffer = nil

	// Close error log if open
	if cb.ErrorLog != nil {
		File4Close(cb.ErrorLog)
		cb.ErrorLog = nil
	}

	return ErrorNone
}

// Code4Close closes all open data files associated with a CODE4 structure.
// This mirrors the code4close function from the CodeBase library.
//
// The function safely iterates through all open data files and closes them,
// including any associated memo files. Files are removed from the internal
// data file list during the process.
//
// Returns ErrorNone on success, ErrorMemory if cb is nil.
func Code4Close(cb *Code4) int {
	if cb == nil {
		return ErrorMemory
	}

	// Safely close all data files in the list
	for cb.DataFileList.NumLinks > 0 {
		current := list4First(&cb.DataFileList)
		if current == nil {
			break // Safety check
		}

		// Remove from list first to avoid issues
		list4Remove(&cb.DataFileList, current)

		// Try to get the Data4 from its embedded Link4
		// Note: We use Data4 not Data4File since that's what gets added to DataFileList
		data := data4FromLink(current)
		if data != nil && data.DataFile != nil {
			// Close the file safely
			File4Close(&data.DataFile.File)

			// Close memo file if open
			if data.DataFile.MemoFile != nil {
				File4Close(&data.DataFile.MemoFile.File)
			}
		}
	}

	cb.ErrorCode = ErrorNone
	return ErrorNone
}

// Code4Exit performs cleanup and exits the application.
// This mirrors the code4exit function from the CodeBase library.
//
// The function calls Code4InitUndo to clean up all resources and then
// exits the application with status code 1. This is typically used
// for fatal error conditions.
func Code4Exit(cb *Code4) {
	if cb != nil {
		Code4InitUndo(cb)
		os.Exit(1)
	}
}

// Code4DateFormat returns the current date format string.
// This mirrors the code4dateFormat function from the CodeBase library.
//
// Currently returns the default format "MM/DD/YY". Future versions
// may support configurable date formats.
//
// Returns the date format string, or default if cb is nil.
func Code4DateFormat(cb *Code4) string {
	if cb == nil {
		return "MM/DD/YY" // Default format
	}
	// TODO: Implement date format storage in Code4
	return "MM/DD/YY"
}

// Code4DateFormatSet sets the date format for the CODE4 structure.
// This mirrors the code4dateFormatSet function from the CodeBase library.
//
// Currently a placeholder implementation that accepts any format string
// but does not enforce validation. Future versions will implement
// full date format validation and storage.
//
// Returns ErrorNone on success, ErrorMemory if cb is nil.
func Code4DateFormatSet(cb *Code4, format string) int {
	if cb == nil {
		return ErrorMemory
	}
	// TODO: Implement date format validation and storage
	return ErrorNone
}

// Code4Data finds an open data file by its alias name.
// This mirrors the code4data function from the CodeBase library.
//
// The function searches through all currently open data files in the
// CODE4 structure's data file list and returns the first match found.
// The alias comparison is case-insensitive.
//
// Returns a pointer to the Data4 structure if found, nil otherwise.
func Code4Data(cb *Code4, alias string) *Data4 {
	if cb == nil || alias == "" {
		return nil
	}

	// Traverse DataFileList to find matching alias
	current := list4First(&cb.DataFileList)
	for current != nil {
		// Get the Data4 from its embedded Link4
		data := data4FromLink(current)
		if data != nil && strings.EqualFold(data.Alias, alias) {
			return data
		}

		// Move to next
		current = list4Next(&cb.DataFileList, current)
		if current == list4First(&cb.DataFileList) {
			break // Circular list, we've come back to start
		}
	}

	return nil
}

// Code4IndexExtension returns the current index file extension.
// This mirrors the code4indexExtension function from the CodeBase library.
//
// The function returns the file extension used for index files,
// typically "CDX" for Visual FoxPro compatibility.
//
// Returns the index extension string, "CDX" if cb is nil.
func Code4IndexExtension(cb *Code4) string {
	if cb == nil {
		return "CDX"
	}
	return string(cb.IndexExtension[:3])
}

// setError sets error code in CODE4 structure
func setError(cb *Code4, errorCode int) int {
	if cb != nil {
		cb.ErrorCode = errorCode
	}
	return errorCode
}

// List management functions for Link4 and List4

// list4Add adds a link to a list
func list4Add(list *List4, link *Link4) {
	if list == nil || link == nil {
		return
	}

	if list.LastNode == nil {
		// First node in list
		list.LastNode = link
		list.Selected = link
		link.Next = link
		link.Prev = link
	} else {
		// Insert at end
		lastNode := list.LastNode
		firstNode := lastNode.Next

		link.Next = firstNode
		link.Prev = lastNode
		lastNode.Next = link
		firstNode.Prev = link

		list.LastNode = link
	}

	list.NumLinks++
}

// list4Remove removes a link from a list
func list4Remove(list *List4, link *Link4) {
	if list == nil || link == nil || list.NumLinks == 0 {
		return
	}

	if list.NumLinks == 1 {
		// Only node in list
		list.LastNode = nil
		list.Selected = nil
	} else {
		// Update neighboring links
		link.Prev.Next = link.Next
		link.Next.Prev = link.Prev

		// Update list pointers if necessary
		if list.LastNode == link {
			list.LastNode = link.Prev
		}
		if list.Selected == link {
			list.Selected = link.Next
		}
	}

	// Clear the removed link
	link.Next = nil
	link.Prev = nil

	list.NumLinks--
}

// list4First returns first link in list
func list4First(list *List4) *Link4 {
	if list == nil || list.LastNode == nil {
		return nil
	}
	return list.LastNode.Next
}

// list4Last returns last link in list
//
//nolint:unused // Future use
func list4Last(list *List4) *Link4 {
	if list == nil {
		return nil
	}
	return list.LastNode
}

// list4Next returns next link after current
func list4Next(list *List4, current *Link4) *Link4 {
	if list == nil || current == nil {
		return nil
	}
	return current.Next
}

// container_of gets the parent structure from an embedded Link4
// This mimics the C macro container_of
//
//nolint:unused // Future use
func data4FileFromLink(link *Link4) *Data4File {
	if link == nil {
		return nil
	}
	// Calculate offset of Link field in Data4File structure
	// Link is the first field, so offset is 0
	return (*Data4File)(unsafe.Pointer(link))
}

func data4FromLink(link *Link4) *Data4 {
	if link == nil {
		return nil
	}
	// Calculate offset of Link field in Data4 structure
	// Link is the first field, so offset is 0
	return (*Data4)(unsafe.Pointer(link))
}

// File4Close closes a file handle and releases associated resources.
// This mirrors the file4close function from the CodeBase library.
//
// The function safely closes the file handle, cleans up any locks,
// and removes temporary files if applicable. It's safe to call
// multiple times on the same file handle.
//
// Returns ErrorNone on success, ErrorMemory if f4 is nil,
// ErrorClose if the file close operation fails.
func File4Close(f4 *File4) int {
	if f4 == nil {
		return ErrorMemory
	}

	// Add safety check to prevent segmentation fault
	if f4.Handle != nil {
		// Clean up any locks associated with this file
		CleanupLocks(f4)

		// Only close if the handle is actually open
		if f4.FileCreated {
			err := f4.Handle.Close()
			f4.Handle = nil
			f4.FileCreated = false
			if err != nil {
				return ErrorClose
			}
		} else {
			// Handle was not properly created, just clear it
			f4.Handle = nil
		}
	}

	// Clean up temporary files
	if f4.IsTemp && f4.Name != "" {
		os.Remove(f4.Name)
	}

	return ErrorNone
}

// File4Create creates a new file with the specified name.
// This mirrors the file4create function from the CodeBase library.
//
// The function creates a new file, truncating it if it already exists
// (unless safety mode prevents overwriting). The file is opened in
// read-write mode and initialized with default parameters.
//
// Parameters:
//   - f4: File4 structure to initialize
//   - cb: CODE4 context for error handling and settings
//   - fileName: Path and name of file to create
//   - doAdvance: Currently unused (for CodeBase compatibility)
//
// Returns ErrorNone on success, ErrorMemory for nil parameters,
// ErrorCreate if file creation fails or safety prevents overwrite.
func File4Create(f4 *File4, cb *Code4, fileName string, doAdvance int) int {
	if f4 == nil || cb == nil || fileName == "" {
		return setError(cb, ErrorMemory)
	}

	// Check if file exists and safety is on
	if cb.Safety != 0 {
		if _, err := os.Stat(fileName); err == nil {
			return setError(cb, ErrorCreate) // File exists
		}
	}

	// Create the file
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return setError(cb, ErrorCreate)
	}

	// Initialize File4 structure
	f4.Handle = file
	f4.Name = fileName
	f4.IsReadOnly = false
	f4.IsTemp = cb.CreateTemp
	f4.Length = 0
	f4.FileCreated = true
	f4.AccessMode = AccessDenyNone

	return setError(cb, ErrorNone)
}

// File4Open opens an existing file with the specified access mode.
// This mirrors the file4open function from the CodeBase library.
//
// The function opens an existing file and initializes the File4 structure
// with appropriate settings based on the access mode.
//
// Parameters:
//   - f4: File4 structure to initialize
//   - cb: CODE4 context for error handling
//   - fileName: Path and name of file to open
//   - accessMode: File access mode (AccessDenyNone, AccessDenyRW, etc.)
//
// Returns ErrorNone on success, ErrorMemory for nil parameters,
// ErrorOpen if file cannot be opened or accessed.
func File4Open(f4 *File4, cb *Code4, fileName string, accessMode int) int {
	if f4 == nil || cb == nil || fileName == "" {
		return setError(cb, ErrorMemory)
	}

	// Determine open mode
	var flag int
	switch accessMode {
	case AccessDenyRW:
		flag = os.O_RDONLY
	case AccessDenyNone:
		flag = os.O_RDWR
	default:
		flag = os.O_RDWR
	}

	file, err := os.OpenFile(fileName, flag, 0644)
	if err != nil {
		return setError(cb, ErrorOpen)
	}

	// Get file info
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return setError(cb, ErrorOpen)
	}

	// Initialize File4 structure
	f4.Handle = file
	f4.Name = fileName
	f4.IsReadOnly = (accessMode == AccessDenyRW)
	f4.IsTemp = false
	f4.Length = info.Size()
	f4.FileCreated = true
	f4.AccessMode = accessMode

	return setError(cb, ErrorNone)
}

// File4Read reads data from a file at the specified position.
// This mirrors the file4read function from the CodeBase library.
//
// The function seeks to the specified position and reads up to len bytes
// into the provided buffer.
//
// Parameters:
//   - f4: File4 structure representing the open file
//   - pos: File position to start reading from
//   - buffer: Buffer to read data into
//   - len: Maximum number of bytes to read
//
// Returns the actual number of bytes read, 0 on error.
func File4Read(f4 *File4, pos File4Long, buffer []byte, len uint32) uint32 {
	if f4 == nil || f4.Handle == nil || buffer == nil {
		return 0
	}

	// Seek to position
	if _, err := f4.Handle.Seek(pos, 0); err != nil {
		return 0
	}

	// Read data
	n, err := f4.Handle.Read(buffer[:len])
	if err != nil {
		return 0
	}

	return uint32(n)
}

// File4Write writes data to a file at the specified position.
// This mirrors the file4write function from the CodeBase library.
//
// The function seeks to the specified position and writes len bytes
// from the buffer to the file. The file length is updated if the
// write extends beyond the current end of file.
//
// Parameters:
//   - f4: File4 structure representing the open file
//   - pos: File position to start writing to
//   - buffer: Buffer containing data to write
//   - len: Number of bytes to write
//
// Returns ErrorNone on success, ErrorMemory for nil parameters,
// ErrorWrite for write failures, ErrorSeek for positioning errors.
func File4Write(f4 *File4, pos File4Long, buffer []byte, len uint32) int {
	if f4 == nil || f4.Handle == nil || buffer == nil {
		return ErrorMemory
	}

	if f4.IsReadOnly {
		return ErrorWrite
	}

	// Seek to position
	if _, err := f4.Handle.Seek(pos, 0); err != nil {
		return ErrorSeek
	}

	// Write data
	n, err := f4.Handle.Write(buffer[:len])
	if err != nil || uint32(n) != len {
		return ErrorWrite
	}

	// Update file length
	if pos+int64(len) > f4.Length {
		f4.Length = pos + int64(len)
	}

	return ErrorNone
}

// File4Length returns the current length of a file.
// This mirrors the file4len function from the CodeBase library.
//
// Returns the file length in bytes, 0 if f4 is nil.
func File4Length(f4 *File4) File4Long {
	if f4 == nil {
		return 0
	}
	return f4.Length
}

// File4Flush flushes any buffered writes to disk.
// This mirrors the file4flush function from the CodeBase library.
//
// The function ensures all pending writes are committed to the
// underlying storage device.
//
// Returns ErrorNone on success, ErrorMemory if f4 is nil,
// ErrorWrite if the flush operation fails.
func File4Flush(f4 *File4) int {
	if f4 == nil || f4.Handle == nil {
		return ErrorMemory
	}

	if err := f4.Handle.Sync(); err != nil {
		return ErrorWrite
	}

	return ErrorNone
}

// File4Truncate truncates a file to the specified size.
// This mirrors the file4truncate function from the CodeBase library.
//
// The function changes the file size to the specified number of bytes.
// If the file is larger than the specified size, the excess data is
// discarded. If smaller, the behavior is system-dependent.
//
// Parameters:
//   - f4: File4 structure representing the file to truncate
//   - size: New file size in bytes
//
// Returns ErrorNone on success, ErrorMemory if f4 is nil,
// ErrorWrite if the file is read-only or truncation fails.
func File4Truncate(f4 *File4, size File4Long) int {
	if f4 == nil || f4.Handle == nil {
		return ErrorMemory
	}

	if f4.IsReadOnly {
		return ErrorWrite
	}

	// Truncate the file
	if err := f4.Handle.Truncate(size); err != nil {
		return ErrorWrite
	}

	// Update file length
	f4.Length = size

	return ErrorNone
}

// Helper function to construct file paths with extension
func constructPath(baseName, extension string) string {
	if filepath.Ext(baseName) == "" {
		return baseName + "." + strings.ToUpper(extension)
	}
	return baseName
}
