package deepresearch

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// parsePDFOutput is the JSON structure emitted by python/parse_pdf.py.
type parsePDFOutput struct {
	Companies []SupplyChainEntity `json:"companies"`
	Error     string              `json:"error,omitempty"`
}

// scriptPath returns the absolute path to python/parse_pdf.py, resolved
// relative to this source file so it works regardless of working directory.
func scriptPath() string {
	_, file, _, _ := runtime.Caller(0)
	// file = .../internal/deepresearch/parser.go  →  go up two dirs to repo root
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "python", "parse_pdf.py")
}

// ParseAnnualReport runs the Python PDF parser on pdfURL and returns the
// extracted supply chain entities found in the Related Party Transactions section.
func ParseAnnualReport(pdfURL string) ([]SupplyChainEntity, error) {
	cmd := exec.Command("python3", scriptPath(), pdfURL)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("pdf parser: %s", exitErr.Stderr)
		}
		return nil, fmt.Errorf("pdf parser: %w", err)
	}

	var result parsePDFOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("pdf parser output: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("pdf parser: %s", result.Error)
	}
	return result.Companies, nil
}
