package graph

import (
	"path/filepath"
	"strings"

	"ctxengine/internal/parse"
	"ctxengine/internal/walker"
)

// ParsedRepo holds parsed results per file, keyed by repo-relative path.
type ParsedRepo map[string]parse.ParsedFile

// Build constructs a Graph from a walker.Result using the given parsers.
// It resolves import paths to in-repo file nodes where possible and adds
// call edges between symbol nodes. Parsing is done concurrently via a
// worker pool (see parse.ParseConcurrently).
func Build(result *walker.Result, parsers []parse.Parser) (*Graph, ParsedRepo, error) {
	g := New()

	// Step 1: Add all file nodes (sequential — graph mutation is not thread-safe).
	for _, f := range result.Files {
		g.AddNode(Node{ID: f.RelPath, Kind: NodeFile, Path: f.RelPath})
	}

	// Step 2: Parse all files concurrently.
	inputs := make([]parse.FileInput, len(result.Files))
	for i, f := range result.Files {
		inputs[i] = parse.FileInput{Path: f.RelPath, Content: f.Content}
	}
	parsedMap := parse.ParseConcurrently(inputs, parsers, 8)
	parsed := make(ParsedRepo, len(parsedMap))
	for k, v := range parsedMap {
		parsed[k] = v
	}

	// Step 3: Add symbol nodes (sequential — graph mutation).
	for _, pf := range parsed {
		for _, sym := range pf.Symbols {
			symID := pf.Path + "#" + sym.Name
			kind := NodeFunction
			if sym.Kind == "type" {
				kind = NodeType
			}
			g.AddNode(Node{ID: symID, Kind: kind, Path: pf.Path})
		}
	}

	// Build a package→file index so import paths can be resolved to file nodes.
	// For Go: the module path is trimmed to find the relative package path.
	pkgToFile := buildPackageIndex(parsed)

	// Step 2: Add import and call edges.
	for _, pf := range parsed {
		for _, imp := range pf.Imports {
			// Resolve import string to a file node in the repo.
			target := resolveImport(imp, pkgToFile)
			if target == "" {
				continue // external dependency, skip
			}
			g.MergeEdge(Edge{From: pf.Path, To: target, Kind: EdgeImports, Weight: 1.0})
		}

		for _, callRef := range pf.Calls {
			// Resolve "pkg.Symbol" or "Symbol" to a symbol node.
			target := resolveCall(callRef, pf.Package, pkgToFile, parsed)
			if target == "" {
				continue
			}
			g.MergeEdge(Edge{From: pf.Path, To: target, Kind: EdgeCalls, Weight: 1.0})
		}
	}

	return g, parsed, nil
}

// buildPackageIndex maps Go package import paths (as declared) to repo-relative
// file paths. For multi-file packages, the first file encountered wins for the
// file-level mapping; symbol-level resolution checks all files in the package.
func buildPackageIndex(parsed ParsedRepo) map[string][]string {
	// pkgPath → []filePaths
	idx := make(map[string][]string)
	for path, pf := range parsed {
		if pf.Package == "" {
			continue
		}
		// Derive the "package path" from the file's directory, e.g.
		// "internal/payments/retry.go" → package path segment "internal/payments"
		dir := filepath.Dir(path)
		if dir == "." {
			dir = ""
		}
		idx[dir] = append(idx[dir], path)

		// Also index by package name alone as a fallback.
		idx[pf.Package] = append(idx[pf.Package], path)
	}
	return idx
}

// resolveImport maps a raw import string to a repo-relative file path.
// It strips the module prefix (anything up to the last known directory component)
// and does a best-effort match against the package index.
func resolveImport(imp string, pkgToFile map[string][]string) string {
	// Direct match on full import path (works for relative packages).
	if files, ok := pkgToFile[imp]; ok && len(files) > 0 {
		return files[0]
	}
	// Strip leading module prefix by trying progressively shorter suffixes.
	// e.g. "ctxengine/internal/walker" → try "internal/walker", "walker".
	parts := strings.Split(imp, "/")
	for i := 1; i < len(parts); i++ {
		suffix := strings.Join(parts[i:], "/")
		if files, ok := pkgToFile[suffix]; ok && len(files) > 0 {
			return files[0]
		}
	}
	return ""
}

// resolveCall maps a call reference ("pkg.Symbol" or "Symbol") to a symbol
// node ID in the graph, using the package index.
func resolveCall(ref, callerPkg string, pkgToFile map[string][]string, parsed ParsedRepo) string {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) == 2 {
		pkgName, symName := parts[0], parts[1]
		// Find files that belong to pkgName.
		files, ok := pkgToFile[pkgName]
		if !ok {
			return ""
		}
		for _, f := range files {
			pf := parsed[f]
			for _, sym := range pf.Symbols {
				if sym.Name == symName {
					return f + "#" + symName
				}
			}
		}
	} else {
		// Unqualified call: check the same package.
		symName := parts[0]
		files := pkgToFile[callerPkg]
		for _, f := range files {
			pf := parsed[f]
			for _, sym := range pf.Symbols {
				if sym.Name == symName {
					return f + "#" + symName
				}
			}
		}
	}
	return ""
}
