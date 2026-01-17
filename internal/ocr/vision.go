package ocr

import (
	"encoding/json"
	"fmt"
	"os/exec"
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
import sys
try:
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
except ImportError:
    # Fallback: try direct Vision framework access via pyobjc
    import objc
    from Foundation import NSURL
    from Quartz import CIImage
    import Vision
    
    image_url = NSURL.fileURLWithPath_('%s')
    request = Vision.VNRecognizeTextRequest.alloc().init()
    request.setRecognitionLevel_(1 if '%s' == 'accurate' else 0)
    
    handler = Vision.VNImageRequestHandler.alloc().initWithURL_options_(image_url, None)
    success = handler.performRequests_error_([request], None)
    
    results = request.results()
    output = []
    if results:
        for observation in results:
            text = observation.topCandidates_(1)[0].string()
            confidence = observation.confidence()
            bbox = observation.boundingBox()
            output.append({
                "text": text,
                "confidence": float(confidence),
                "bounding_box": [bbox.origin.x, bbox.origin.y, bbox.size.width, bbox.size.height]
            })
    print(json.dumps(output))
`, imagePath, e.recognitionLevel, e.language, imagePath, e.recognitionLevel)

	cmd := exec.Command("python3", "-c", script)
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
