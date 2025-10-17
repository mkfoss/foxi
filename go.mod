module github.com/mkfoss/foxi

go 1.21

// Foxi - High Level Go DBF Package
// Supports both CGO (mkfdbf) and pure Go (gomkfdbf) backends via build tags
// Default: pure Go backend
// CGO backend: build with -tags foxicgo
