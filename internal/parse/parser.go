package parse

// Symbol represents an exported function, type, or constant declared in a file.
type Symbol struct {
	Name      string `json:"name"`
	Signature string `json:"signature"` // e.g. "func Foo(x int) error" or "type Bar struct"
	Kind      string `json:"kind"`      // "func", "type", "const", "var"
}

// ParsedFile holds the analysis results for a single source file.
type ParsedFile struct {
	// Path is the repo-relative path of the file.
	Path string `json:"path"`
	// Package is the declared package name (language-specific, empty if N/A).
	Package string `json:"package"`
	// Imports are raw import strings as they appear in the source (e.g. Go import paths).
	Imports []string `json:"imports"`
	// Calls is the set of symbols called/referenced from within this file
	// that could resolve to in-repo symbols (best-effort; format: "pkg.Symbol" or "Symbol").
	Calls []string `json:"calls"`
	// Symbols are the top-level declarations exported by this file.
	Symbols []Symbol `json:"symbols"`
}

// Parser extracts structural information from source files of a particular language.
type Parser interface {
	// Extensions returns the file extensions this parser handles (e.g. [".go"]).
	Extensions() []string
	// Parse analyses the given file content and returns its structural information.
	Parse(path, content string) (ParsedFile, error)
	// ExtractSignatures returns only exported symbol signatures for a file,
	// stopping before function bodies. Used by the budget packer for signature_only mode.
	ExtractSignatures(path, content string) ([]Symbol, error)
}
