package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hetulpatel/Arbitrage/internal/llm"
	"github.com/hetulpatel/Arbitrage/internal/matches"
)

const systemPrompt = "You are a strict arbitrage validator. Determine if two binary markets resolve identically with no ambiguity. Reject if timing, definitions, or data sources differ. Respond only with JSON."

// Service validates market pairs via LLM.
type Service struct {
	llm          *llm.Client
	pdfExtractor PDFExtractor
	systemPrompt string
}

// NewService creates a validator.
func NewService(cfg Config) (*Service, error) {
	if cfg.LLMClient == nil {
		return nil, fmt.Errorf("validator: llm client is required")
	}
	system := cfg.SystemPrompt
	if strings.TrimSpace(system) == "" {
		system = systemPrompt
	}
	return &Service{
		llm:          cfg.LLMClient,
		pdfExtractor: cfg.PDFExtractor,
		systemPrompt: system,
	}, nil
}

// Validate runs the LLM prompt and returns the verdict.
func (s *Service) Validate(ctx context.Context, payload *matches.Payload) (*Result, error) {
	if s == nil {
		return nil, fmt.Errorf("validator: service is nil")
	}
	if payload == nil {
		return nil, fmt.Errorf("validator: payload is nil")
	}

	promptInput, err := buildPromptPayload(ctx, payload, s.pdfExtractor)
	if err != nil {
		return nil, err
	}
	inputJSON, err := json.MarshalIndent(promptInput, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("validator: marshal prompt input: %w", err)
	}

	userPrompt := strings.Join([]string{
		"Compare the following Polymarket and Kalshi markets. Polymarket and Kalshi are prediction markets, you are helping with an arbitrage detection system.",
		"Right now, a possibile risk-free arbitrage is possible if the two markets resolve identically.",
		"They must represent the exact same binary outcome, their resolution criteria must be the same, and have matching cutoff/resolution criteria to be valid.",
		"For example, they can have different resolution sources, but as long as the criteria and the resolution sources agree on the exact definition, that is valid.",
		"If either market allows outcomes not strictly YES/NO for the exact same event, answer false. If a potential resolution where yes or no are not the only possibilities, answer false.",
		"Pay special attention to timing, settlement sources, definitions, tiebreakers, cancellations, or alternate clauses.",
		"If unsure, treat it as invalid. Answer concisely with only necessary information, nothing too much more.",
		"Return EXACTLY this JSON format:\n{\n  \"ValidResolution\": true|false,\n  \"ResolutionReason\": \"short explanation\"\n}\n\nInput JSON:\n" + string(inputJSON),
	}, "\n")

	raw, err := s.llm.Complete(ctx, s.systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("validator: llm call: %w", err)
	}

	res, err := parseResult(raw)
	if err != nil {
		return nil, fmt.Errorf("validator: parse response: %w", err)
	}
	return res, nil
}
