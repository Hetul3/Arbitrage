package validator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const (
	defaultPDFTimeout = 25 * time.Second
	maxPDFBytes       = 8 << 20 // 8 MB
)

// CommandPDFExtractor downloads a PDF and converts it to text via pdftotext.
type CommandPDFExtractor struct {
	client  *http.Client
	binary  string
	timeout time.Duration
}

// NewCommandPDFExtractor returns an extractor using the pdftotext CLI.
func NewCommandPDFExtractor(bin string) *CommandPDFExtractor {
	if bin == "" {
		bin = os.Getenv("PDFTOTEXT_BIN")
	}
	if bin == "" {
		bin = "pdftotext"
	}
	return &CommandPDFExtractor{
		client: &http.Client{
			Timeout: defaultPDFTimeout,
		},
		binary:  bin,
		timeout: defaultPDFTimeout,
	}
}

// Extract downloads the PDF and returns the extracted text.
func (e *CommandPDFExtractor) Extract(ctx context.Context, rawURL string) (string, error) {
	if e == nil {
		return "", fmt.Errorf("pdf extractor is nil")
	}
	if rawURL == "" {
		return "", fmt.Errorf("pdf url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("pdf download failed: %s", resp.Status)
	}

	tmpPDF, err := os.CreateTemp("", "contract-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpPDF.Name())

	limited := io.LimitReader(resp.Body, maxPDFBytes)
	if _, err := io.Copy(tmpPDF, limited); err != nil {
		tmpPDF.Close()
		return "", err
	}
	tmpPDF.Close()

	tmpTxtFile, err := os.CreateTemp("", "contract-*.txt")
	if err != nil {
		return "", err
	}
	tmpTxt := tmpTxtFile.Name()
	tmpTxtFile.Close()
	defer os.Remove(tmpTxt)

	cmdCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, e.binary, "-layout", tmpPDF.Name(), tmpTxt)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}

	data, err := os.ReadFile(tmpTxt)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
