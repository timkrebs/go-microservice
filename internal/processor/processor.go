package processor

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/timkrebs/image-processor/internal/models"
)

// Processor handles image processing operations
type Processor struct{}

// New creates a new image processor
func New() *Processor {
	return &Processor{}
}

// ProcessResult contains the processed image and metadata
type ProcessResult struct {
	Data        []byte
	ContentType string
	Width       int
	Height      int
}

// Process applies the given operations to an image
func (p *Processor) Process(reader io.Reader, contentType string, operations []models.Operation) (*ProcessResult, error) {
	// Decode the image
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Convert to NRGBA for processing
	nrgba := imaging.Clone(img)

	// Apply each operation
	for _, op := range operations {
		nrgba, err = p.applyOperation(nrgba, op)
		if err != nil {
			return nil, fmt.Errorf("failed to apply operation %s: %w", op.Operation, err)
		}
	}

	// Encode the result
	var buf bytes.Buffer
	outputContentType := contentType

	switch {
	case strings.Contains(contentType, "png"), format == "png":
		if err := png.Encode(&buf, nrgba); err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
		outputContentType = "image/png"
	default:
		if err := jpeg.Encode(&buf, nrgba, &jpeg.Options{Quality: 90}); err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
		outputContentType = "image/jpeg"
	}

	bounds := nrgba.Bounds()
	return &ProcessResult{
		Data:        buf.Bytes(),
		ContentType: outputContentType,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
	}, nil
}

func (p *Processor) applyOperation(img *image.NRGBA, op models.Operation) (*image.NRGBA, error) {
	switch op.Operation {
	case models.OperationResize:
		return p.resize(img, op.Parameters)
	case models.OperationThumbnail:
		return p.thumbnail(img, op.Parameters)
	case models.OperationBlur:
		return p.blur(img, op.Parameters)
	case models.OperationSharpen:
		return p.sharpen(img, op.Parameters)
	case models.OperationGrayscale:
		return p.grayscale(img)
	case models.OperationSepia:
		return p.sepia(img)
	case models.OperationRotate:
		return p.rotate(img, op.Parameters)
	case models.OperationFlip:
		return p.flip(img, op.Parameters)
	case models.OperationBrightness:
		return p.brightness(img, op.Parameters)
	case models.OperationContrast:
		return p.contrast(img, op.Parameters)
	case models.OperationSaturation:
		return p.saturation(img, op.Parameters)
	default:
		return nil, fmt.Errorf("unknown operation: %s", op.Operation)
	}
}

func (p *Processor) getIntParam(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return defaultVal
}

func (p *Processor) getFloatParam(params map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		}
	}
	return defaultVal
}

func (p *Processor) getBoolParam(params map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if val, ok := v.(bool); ok {
			return val
		}
	}
	return defaultVal
}
