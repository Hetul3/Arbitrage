package validator

import (
	"context"

	"github.com/hetulpatel/Arbitrage/internal/llm"
)

// Result represents the structured LLM verdict.
type Result struct {
	ValidResolution  bool   `json:"ValidResolution"`
	ResolutionReason string `json:"ResolutionReason"`
}

// Config controls the validator behavior.
type Config struct {
	LLMClient    *llm.Client
	PDFExtractor PDFExtractor
	SystemPrompt string
}

// PDFExtractor fetches PDF text from a URL.
type PDFExtractor interface {
	Extract(ctx context.Context, url string) (string, error)
}
