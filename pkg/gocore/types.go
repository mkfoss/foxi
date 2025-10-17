// Package pkg provides core types and structures for the gomkfdbf library
// Direct translation from CodeBase C library structures
package pkg

import (
	"os"
	"time"
)

// Constants matching C definitions
const (
	// Field type constants (from C)
	FieldTypeChar     = 'C' // Character field
	FieldTypeNumeric  = 'N' // Numeric field  
	FieldTypeFloat    = 'F' // Float field
	FieldTypeDate     = 'D' // Date field
	FieldTypeLogical  = 'L' // Logical field
	FieldTypeMemo     = 'M' // Memo field
	FieldTypeGeneral  = 'G' // General/OLE field
	FieldTypePicture  = 'P' // Picture field
	FieldTypeCurrency = 'Y' // Currency field
	FieldTypeDateTime = 'T' // DateTime field
	FieldTypeInteger  = 'I' // Integer field
	FieldTypeVarChar  = 'V' // VarChar field

	// Access mode constants
	AccessDenyRW   = 0x10 // Deny read/write
	AccessDenyNone = 0x40 // Deny none (shared)

	// Error codes (matching C definitions)
	ErrorNone     = 0
	ErrorMemory   = -910
	ErrorOpen     = -920
	ErrorRead     = -930
	ErrorWrite    = -940
	ErrorSeek     = -950
	ErrorClose    = -960
	ErrorCreate   = -970
	ErrorData     = -980
	ErrorIndex    = -990

	// Path and name lengths
	MaxPathLen     = 260
	MaxFieldName   = 10
	MaxDateFormat  = 20
	MaxUserIdLen   = 8
	MaxNetIdLen    = 8

	// Return codes (r4 constants)
	R4Success  = 0  // Operation successful
	R4Found    = 1  // Primary key match
	R4After    = 2  // Position after
	R4Eof      = 3  // End of file
	R4Bof      = 4  // Beginning of file
	R4Unique   = 20 // Key is not unique

	// Field type constants for creation
	R4Num = 'N' // Numeric field type

	// Access modes 
	Open4DenyRW   = AccessDenyRW
	Open4DenyNone = AccessDenyNone
)

// Link4 represents a doubly-linked list node (from LINK4 in C)
type Link4 struct {
	Next *Link4
	Prev *Link4
}

// List4 represents a doubly-linked list (from LIST4 in C)  
type List4 struct {
	LastNode *Link4
	Selected *Link4
	NumLinks uint16
}

// File4Long represents file position/length (from FILE4LONG in C)
type File4Long = int64

// File4 represents file handle and operations (from FILE4 in C)
type File4 struct {
	Handle       *os.File
	Name         string
	IsReadOnly   bool
	IsTemp       bool
	Length       File4Long
	DoBuffer     bool
	AccessMode   int
	WriteBuffer  bool
	FileCreated  bool
	ExpectedSize int64
}

// Currency4 represents currency data type (from CURRENCY4 in C)
type Currency4 struct {
	Lo [4]uint16
}

// Field4Info represents field definition for table creation (from FIELD4INFO in C)
type Field4Info struct {
	Name   string
	Type   rune   // Field type character
	Length uint16
	Dec    uint16 // Decimal places for numeric fields
	Nulls  uint16 // Allow nulls flag
}

// Field4Image represents the raw field structure in DBF header (from FIELD4IMAGE in C)
type Field4Image struct {
	Name      [11]byte  // Field name (null-terminated)
	Type      byte      // Field type
	Offset    int32     // Field offset in record
	Length    byte      // Field length
	Dec       byte      // Decimal places
	NullBinary byte     // Null/binary flags (Fox 3.0+)
	Filler2   [12]byte  // Reserved space
	HasTag    byte      // Has production index tag
}

// Field4 represents a field in a database table (from FIELD4 in C)
type Field4 struct {
	Name     [11]byte   // Field name
	Length   uint16     // Field length
	Dec      uint16     // Decimal places
	Type     int16      // Field type
	Offset   uint32     // Offset in record
	Data     *Data4     // Pointer to parent DATA4
	Null     byte       // Null support flag
	NullBit  uint16     // Null bit mask
	Binary   byte       // Binary field flag
	Memo     *F4Memo    // Memo field handler
}

// F4Memo represents memo field data (from F4MEMO in C)  
type F4Memo struct {
	IsChanged bool
	Status    int
	Contents  []byte
	Length    uint32
	MaxLength uint32
	Field     *Field4
}

// DbfHeader represents the DBF file header structure
type DbfHeader struct {
	Version   byte      // DBF version
	Year      byte      // Last update year (YY)
	Month     byte      // Last update month  
	Day       byte      // Last update day
	NumRecs   int32     // Number of records
	HeaderLen uint16    // Header length
	RecordLen uint16    // Record length
	Reserved  [16]byte  // Reserved bytes
	// Additional fields for FoxPro compatibility would follow
}

// Memo4Header represents memo file header (from MEMO4HEADER in C)
type Memo4Header struct {
	NextBlock int32     // Next available block
	Unused    [2]byte   // Unused
	BlockSize int16     // Block size in bytes
}

// Memo4File represents memo file handle (from MEMO4FILE in C)
type Memo4File struct {
	File      File4
	BlockSize int16
	Data      *Data4File
	FileLock  int
}

// Code4 represents the main CodeBase context (from CODE4 in C)
type Code4 struct {
	// Documented members
	AutoOpen            bool      // Automatic production index file opening
	CreateTemp          bool      // Create files as temporary
	ErrDefaultUnique    int16     // Default unique error handling
	ErrExpr             int       // Expression error handling
	ErrFieldName        int       // Field name error handling  
	ErrOff              int       // Show error messages flag
	ErrOpen             int       // File open error handling
	Log                 int16     // Logging enabled
	MemExpandData       int       // Data memory expansion
	MemSizeBuffer       uint32    // Buffer size for pack/zap
	MemSizeSortBuffer   uint32    // Sort buffer size
	MemSizeSortPool     uint32    // Sort pool size
	MemStartData        uint32    // Initial data allocation
	Safety              byte      // File create with safety
	Timeout             int32     // Operation timeout
	Compatibility       int16     // FoxPro compatibility version

	// Internal members
	Initialized         bool      // Initialization flag
	NumericStrLen       int       // Default numeric string length
	Decimals            int       // Default decimal places
	ErrorCode           int       // Last error code
	FieldBuffer         []byte    // Internal field buffer
	IndexExtension      [4]byte   // Index file extension
	DataFileList        List4     // List of open data files
	
	// Transaction support
	TransactionLevel    int         // Current transaction nesting level
	TransactionID       int64       // Unique transaction identifier
	TransactionLog      []*Trans4State // Transaction log for rollback
	ErrorLog            *File4      // Error log file
	CollatingSequence   int       // Collating sequence for VFP
	CodePage            int       // Code page for database creation
}

// Data4File represents the physical database file (from DATA4FILE in C)
type Data4File struct {
	Link        Link4
	File        File4
	Header      DbfHeader
	Fields      []*Field4
	NumFields   int16
	RecordLen   uint16
	MemoFile    *Memo4File
	UserCount   int
	CodeBase    *Code4
	IsValid     bool
}

// Data4 represents a database table instance (from DATA4 in C)  
type Data4 struct {
	Link          Link4
	Alias         string
	CodeBase      *Code4
	Fields        []*Field4
	Indexes       List4
	TagSelected   *Tag4
	CodePage      int
	DataFile      *Data4File
	Record        []byte        // Current record buffer
	RecordOld     []byte        // Previous record buffer  
	RecordBlank   []byte        // Blank record template
	FieldsMemo    []*F4Memo     // Memo fields
	LogVal        int
	TransChanged  byte
	ClientID      int32

	// Navigation state
	recNo         int32         // Current record number
	atEof         bool          // At end of file
	atBof         bool          // At beginning of file
	lastSeekFound bool          // Last seek operation result
}

// Tag4Info represents index tag creation info (from TAG4INFO in C)
type Tag4Info struct {
	Name        string
	Expression  string  
	Filter      string
	Unique      int16
	Descending  uint16
}

// Tag4 represents an index tag (from TAG4 in C)
type Tag4 struct {
	Link      Link4
	Index     *Index4
	TagFile   *Tag4File
	ErrUnique int16     // Unique error handling
	IsValid   bool
	Added     int       // Entry added flag
	Removed   int       // Entry removed flag
}

// Tag4File represents physical tag file (from TAG4FILE in C)
type Tag4File struct {
	Link         Link4
	IndexFile    *Index4File
	Alias        [MaxFieldName + 1]byte // Tag name/alias
	CodeBase     *Code4
	Header       CdxHeader              // CDX header for this tag
	HeaderOffset int32                  // Position in file
	RootWrite    bool                   // Header needs writing
	KeyDec       int                    // Key decimal places
	ExprSource   string                 // Index expression source
	FilterSource string                 // Filter expression source
}

// Index4 represents an index file (from INDEX4 in C)
type Index4 struct {
	Link       Link4
	Tags       List4
	Data       *Data4
	CodeBase   *Code4
	IndexFile  *Index4File
	AccessName [MaxPathLen + 1]byte
	IsValid    bool
}

// Index4File represents physical index file (from INDEX4FILE in C)
type Index4File struct {
	Link      Link4
	Tags      List4
	UserCount int
	CodeBase  *Code4
	DataFile  *Data4File
	File      File4
	IsValid   bool
}

// B4Block represents a B+ tree block for CDX navigation
type B4Block struct {
	BlockNo    int32     // Block number in file
	BlockType  byte      // Block type (leaf, branch, root)
	NumKeys    int16     // Number of keys in block
	KeyLen     int16     // Length of each key
	Data       []byte    // Raw block data
	Keys       []B4Key   // Parsed keys
	Changed    bool      // Block modified flag
	Pointers   []int32   // Child block pointers (for branch blocks)
}

// B4Key represents a key within a B+ tree block
type B4Key struct {
	KeyData   []byte    // Key data
	RecNo     int32     // Record number
	DupCnt    int16     // Duplicate count
	Pointer   int32     // Pointer to child block (for branches)
}

// B4KeyData represents key data with metadata
type B4KeyData struct {
	KeyValue  []byte    // The actual key value
	RecNo     int32     // Record number this key points to
	DupCount  int16     // Number of duplicates
	Trail     byte      // Trailing info
}

// Trans4State represents transaction state for rollback
type Trans4State struct {
	RecNo     int32
	OldRecord []byte
	NewRecord []byte
	Operation int // 1=append, 2=update, 3=delete
	TimeStamp time.Time
}

// Error represents CodeBase error information
type Error struct {
	Code        int
	Description string
	Function    string
	Line        int
	Time        time.Time
}
