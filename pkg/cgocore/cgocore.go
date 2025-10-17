// Package cgocore provides direct access to the mkfdbf C library for DBF file operations.
// This package offers maximum performance through direct C library integration but requires CGO.
//
// Usage:
//   import "github.com/mkfoss/foxi/pkg/cgocore"
//   // Use C library functions directly
//
// For a high-level unified interface that can switch between backends, use the main foxi package instead.
package cgocore

/*
#cgo CFLAGS: -I./mkfdbflib
#cgo LDFLAGS: -L./mkfdbflib -lmkfdbf
#include "d4all.h"
#include <stdlib.h>
*/
import "C"

// This package exposes the mkfdbf C library functionality directly.
// Users can access all C functions through the C import.
//
// Example:
//   cb := (*C.CODE4)(C.malloc(C.sizeof_CODE4))
//   C.code4initLow(cb, nil, 6401, C.long(C.sizeof_CODE4))
//   data := C.d4open(cb, C.CString("data.dbf"))
//
// For easier usage, consider using the main foxi package which provides
// a unified Go-friendly interface that can use this backend via build tags.