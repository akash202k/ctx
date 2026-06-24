package parse_test

import (
	"strings"
	"testing"

	"ctxengine/internal/parse"
)

const sampleGo = `package payments

import (
	"context"
	"ctxengine/internal/graph"
)

type RetryConfig struct {
	MaxAttempts int
	Timeout     int
}

func DoRetry(ctx context.Context, cfg RetryConfig) error {
	graph.New()
	return nil
}

func helper() {}
`

func TestGoParserParse(t *testing.T) {
	p := parse.NewGoParser()
	pf, err := p.Parse("payments/retry.go", sampleGo)
	if err != nil {
		t.Fatal(err)
	}

	if pf.Package != "payments" {
		t.Errorf("expected package payments, got %q", pf.Package)
	}

	wantImports := map[string]bool{
		"context":              true,
		"ctxengine/internal/graph": true,
	}
	for _, imp := range pf.Imports {
		delete(wantImports, imp)
	}
	if len(wantImports) > 0 {
		t.Errorf("missing imports: %v", wantImports)
	}

	hasRetryConfig := false
	hasDoRetry := false
	for _, sym := range pf.Symbols {
		if sym.Name == "RetryConfig" {
			hasRetryConfig = true
		}
		if sym.Name == "DoRetry" {
			hasDoRetry = true
			if !strings.Contains(sym.Signature, "func DoRetry") {
				t.Errorf("unexpected signature: %s", sym.Signature)
			}
		}
	}
	if !hasRetryConfig {
		t.Error("missing RetryConfig symbol")
	}
	if !hasDoRetry {
		t.Error("missing DoRetry symbol")
	}
}

func TestGoParserExtractSignatures(t *testing.T) {
	p := parse.NewGoParser()
	syms, err := p.ExtractSignatures("payments/retry.go", sampleGo)
	if err != nil {
		t.Fatal(err)
	}
	for _, sym := range syms {
		// helper() is unexported — should be absent
		if sym.Name == "helper" {
			t.Error("unexported helper should not appear in signatures")
		}
	}
	if len(syms) == 0 {
		t.Error("expected at least one exported symbol")
	}
}
