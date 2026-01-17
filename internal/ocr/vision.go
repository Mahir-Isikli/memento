package ocr

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type OCRResult struct {
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence"`
	BoundingBox []float64 `json:"bounding_box"`
}

type OCREngine struct {
	recognitionLevel string
	language         string
}

func NewOCREngine() *OCREngine {
	return &OCREngine{
		recognitionLevel: "accurate",
		language:         "en-US",
	}
}

func (e *OCREngine) SetRecognitionLevel(level string) {
	if level == "fast" || level == "accurate" {
		e.recognitionLevel = level
	}
}

func (e *OCREngine) SetLanguage(lang string) {
	e.language = lang
}

func (e *OCREngine) ExtractText(imagePath string) (string, error) {
	results, err := e.Recognize(imagePath)
	if err != nil {
		return "", err
	}
	
	var texts []string
	for _, r := range results {
		texts = append(texts, r.Text)
	}
	return strings.Join(texts, " "), nil
}

func (e *OCREngine) Recognize(imagePath string) ([]OCRResult, error) {
	script := fmt.Sprintf(`
import json
from ocrmac import ocrmac
result = ocrmac.OCR('%s', recognition_level='%s', language_preference=['%s']).recognize()
output = []
for text, confidence, bbox in result:
    output.append({
        "text": text,
        "confidence": confidence,
        "bounding_box": bbox
    })
print(json.dumps(output))
`, imagePath, e.recognitionLevel, e.language)

	// Try to find python in memento's venv first
	pythonPath := "python3"
	execPath, _ := os.Executable()
	if execPath != "" {
		venvPython := filepath.Join(filepath.Dir(execPath), "..", "memento", ".venv", "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			pythonPath = venvPython
		}
	}
	// Also check relative to current working directory
	if cwd, err := os.Getwd(); err == nil {
		venvPython := filepath.Join(cwd, ".venv", "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			pythonPath = venvPython
		}
	}
	// Check in ~/.memento/.venv
	if home, err := os.UserHomeDir(); err == nil {
		venvPython := filepath.Join(home, ".memento", ".venv", "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			pythonPath = venvPython
		}
	}

	cmd := exec.Command(pythonPath, "-c", script)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("OCR failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("OCR failed: %w", err)
	}
	
	var results []OCRResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse OCR output: %w", err)
	}
	
	return results, nil
}

func CheckOCRAvailable() bool {
	script := `
import sys
try:
    from ocrmac import ocrmac
    print("ocrmac")
except ImportError:
    try:
        import Vision
        print("vision")
    except ImportError:
        print("none")
`
	cmd := exec.Command("python3", "-c", script)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	result := strings.TrimSpace(string(output))
	return result == "ocrmac" || result == "vision"
}
