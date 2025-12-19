package processor

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/timkrebs/image-processor/internal/models"
)

// createTestImage creates a simple test image for testing
func createTestImage(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	// Fill with a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}
	return img
}

// encodeTestImage encodes a test image to bytes
func encodeTestImage(t *testing.T, img image.Image, format string) []byte {
	t.Helper()
	var buf bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(&buf, img)
	default:
		t.Fatalf("unsupported format: %s", format)
	}
	if err != nil {
		t.Fatalf("failed to encode image: %v", err)
	}
	return buf.Bytes()
}

func TestNew(t *testing.T) {
	p := New()
	if p == nil {
		t.Error("New() returned nil")
	}
}

func TestProcessor_Process_NoOperations(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", nil)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result == nil {
		t.Fatal("Process() returned nil result")
	}
	if result.ContentType != "image/jpeg" {
		t.Errorf("ContentType = %q, want image/jpeg", result.ContentType)
	}
	if len(result.Data) == 0 {
		t.Error("Data should not be empty")
	}
}

func TestProcessor_Process_PNG(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "png")

	result, err := p.Process(bytes.NewReader(data), "image/png", nil)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.ContentType != "image/png" {
		t.Errorf("ContentType = %q, want image/png", result.ContentType)
	}
}

func TestProcessor_Process_Resize(t *testing.T) {
	p := New()
	img := createTestImage(200, 200)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": 50, "height": 50}},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.Width != 50 {
		t.Errorf("Width = %d, want 50", result.Width)
	}
	if result.Height != 50 {
		t.Errorf("Height = %d, want 50", result.Height)
	}
}

func TestProcessor_Process_Thumbnail(t *testing.T) {
	p := New()
	img := createTestImage(200, 200)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationThumbnail, Parameters: map[string]interface{}{"size": 100}},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.Width != 100 || result.Height != 100 {
		t.Errorf("Dimensions = %dx%d, want 100x100", result.Width, result.Height)
	}
}

func TestProcessor_Process_Blur(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationBlur, Parameters: map[string]interface{}{"sigma": 2.0}},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Data) == 0 {
		t.Error("Blur should produce output")
	}
}

func TestProcessor_Process_Sharpen(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationSharpen, Parameters: map[string]interface{}{"sigma": 1.0}},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Data) == 0 {
		t.Error("Sharpen should produce output")
	}
}

func TestProcessor_Process_Grayscale(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationGrayscale},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Data) == 0 {
		t.Error("Grayscale should produce output")
	}
}

func TestProcessor_Process_Sepia(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationSepia},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Data) == 0 {
		t.Error("Sepia should produce output")
	}
}

func TestProcessor_Process_Rotate(t *testing.T) {
	tests := []struct {
		name  string
		angle float64
	}{
		{"90 degrees", 90},
		{"180 degrees", 180},
		{"45 degrees", 45},
		{"-90 degrees", -90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			img := createTestImage(100, 100)
			data := encodeTestImage(t, img, "jpeg")

			operations := []models.Operation{
				{Operation: models.OperationRotate, Parameters: map[string]interface{}{"angle": tt.angle}},
			}

			result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if len(result.Data) == 0 {
				t.Error("Rotate should produce output")
			}
		})
	}
}

func TestProcessor_Process_FlipHorizontal(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationFlip, Parameters: map[string]interface{}{"horizontal": true}},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Data) == 0 {
		t.Error("Flip should produce output")
	}
}

func TestProcessor_Process_FlipVertical(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationFlip, Parameters: map[string]interface{}{"horizontal": false}},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(result.Data) == 0 {
		t.Error("Flip should produce output")
	}
}

func TestProcessor_Process_Brightness(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
	}{
		{"increase", 20},
		{"decrease", -20},
		{"max", 100},
		{"min", -100},
		{"beyond max", 150},  // should be clamped to 100
		{"beyond min", -150}, // should be clamped to -100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			img := createTestImage(100, 100)
			data := encodeTestImage(t, img, "jpeg")

			operations := []models.Operation{
				{Operation: models.OperationBrightness, Parameters: map[string]interface{}{"amount": tt.amount}},
			}

			result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if len(result.Data) == 0 {
				t.Error("Brightness should produce output")
			}
		})
	}
}

func TestProcessor_Process_Contrast(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
	}{
		{"increase", 30},
		{"decrease", -30},
		{"zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			img := createTestImage(100, 100)
			data := encodeTestImage(t, img, "jpeg")

			operations := []models.Operation{
				{Operation: models.OperationContrast, Parameters: map[string]interface{}{"amount": tt.amount}},
			}

			result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if len(result.Data) == 0 {
				t.Error("Contrast should produce output")
			}
		})
	}
}

func TestProcessor_Process_Saturation(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
	}{
		{"increase", 50},
		{"decrease", -50},
		{"desaturate", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			img := createTestImage(100, 100)
			data := encodeTestImage(t, img, "jpeg")

			operations := []models.Operation{
				{Operation: models.OperationSaturation, Parameters: map[string]interface{}{"amount": tt.amount}},
			}

			result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}

			if len(result.Data) == 0 {
				t.Error("Saturation should produce output")
			}
		})
	}
}

func TestProcessor_Process_MultipleOperations(t *testing.T) {
	p := New()
	img := createTestImage(200, 200)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": 100, "height": 100}},
		{Operation: models.OperationBlur, Parameters: map[string]interface{}{"sigma": 1.0}},
		{Operation: models.OperationBrightness, Parameters: map[string]interface{}{"amount": 10}},
		{Operation: models.OperationGrayscale},
	}

	result, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.Width != 100 || result.Height != 100 {
		t.Errorf("Dimensions = %dx%d, want 100x100", result.Width, result.Height)
	}
	if len(result.Data) == 0 {
		t.Error("Multiple operations should produce output")
	}
}

func TestProcessor_Process_UnknownOperation(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)
	data := encodeTestImage(t, img, "jpeg")

	operations := []models.Operation{
		{Operation: "unknown_op"},
	}

	_, err := p.Process(bytes.NewReader(data), "image/jpeg", operations)
	if err == nil {
		t.Error("Process() should return error for unknown operation")
	}
}

func TestProcessor_Process_InvalidImage(t *testing.T) {
	p := New()

	_, err := p.Process(bytes.NewReader([]byte("not an image")), "image/jpeg", nil)
	if err == nil {
		t.Error("Process() should return error for invalid image data")
	}
}

func TestProcessor_getIntParam(t *testing.T) {
	p := New()

	tests := []struct {
		name       string
		params     map[string]interface{}
		key        string
		defaultVal int
		want       int
	}{
		{"float64 value", map[string]interface{}{"width": float64(100)}, "width", 0, 100},
		{"int value", map[string]interface{}{"width": 200}, "width", 0, 200},
		{"missing key", map[string]interface{}{}, "width", 50, 50},
		{"nil params", nil, "width", 50, 50},
		{"wrong type", map[string]interface{}{"width": "string"}, "width", 30, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.getIntParam(tt.params, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getIntParam() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestProcessor_getFloatParam(t *testing.T) {
	p := New()

	tests := []struct {
		name       string
		params     map[string]interface{}
		key        string
		defaultVal float64
		want       float64
	}{
		{"float64 value", map[string]interface{}{"sigma": float64(2.5)}, "sigma", 0, 2.5},
		{"int value", map[string]interface{}{"sigma": 3}, "sigma", 0, 3.0},
		{"missing key", map[string]interface{}{}, "sigma", 1.5, 1.5},
		{"nil params", nil, "sigma", 1.0, 1.0},
		{"wrong type", map[string]interface{}{"sigma": "string"}, "sigma", 2.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.getFloatParam(tt.params, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getFloatParam() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestProcessor_getBoolParam(t *testing.T) {
	p := New()

	tests := []struct {
		name       string
		params     map[string]interface{}
		key        string
		defaultVal bool
		want       bool
	}{
		{"true value", map[string]interface{}{"horizontal": true}, "horizontal", false, true},
		{"false value", map[string]interface{}{"horizontal": false}, "horizontal", true, false},
		{"missing key", map[string]interface{}{}, "horizontal", true, true},
		{"nil params", nil, "horizontal", false, false},
		{"wrong type", map[string]interface{}{"horizontal": "true"}, "horizontal", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.getBoolParam(tt.params, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getBoolParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessor_resize_ZeroDimensions(t *testing.T) {
	p := New()
	img := createTestImage(100, 100)

	// Both zero - should return original
	result, err := p.resize(img, map[string]interface{}{"width": 0, "height": 0})
	if err != nil {
		t.Fatalf("resize() error = %v", err)
	}

	bounds := result.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("resize(0,0) should return original dimensions, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestProcessor_resize_PreserveAspectRatio(t *testing.T) {
	p := New()
	img := createTestImage(200, 100) // 2:1 aspect ratio

	// Resize with only width specified
	result, err := p.resize(img, map[string]interface{}{"width": 100, "height": 0})
	if err != nil {
		t.Fatalf("resize() error = %v", err)
	}

	bounds := result.Bounds()
	// With height=0, imaging.Resize preserves aspect ratio
	if bounds.Dx() != 100 {
		t.Errorf("Width = %d, want 100", bounds.Dx())
	}
}

func TestProcessor_thumbnail_DefaultSize(t *testing.T) {
	p := New()
	img := createTestImage(200, 200)

	// No size parameter - should use default 150
	result, err := p.thumbnail(img, nil)
	if err != nil {
		t.Fatalf("thumbnail() error = %v", err)
	}

	bounds := result.Bounds()
	if bounds.Dx() != 150 || bounds.Dy() != 150 {
		t.Errorf("thumbnail() default = %dx%d, want 150x150", bounds.Dx(), bounds.Dy())
	}
}

func TestProcessResult_Fields(t *testing.T) {
	result := &ProcessResult{
		ContentType: "image/jpeg",
		Data:        []byte("test data"),
		Width:       100,
		Height:      200,
	}

	if result.ContentType != "image/jpeg" {
		t.Errorf("ContentType = %q, want image/jpeg", result.ContentType)
	}
	if string(result.Data) != "test data" {
		t.Errorf("Data = %q, want test data", string(result.Data))
	}
	if result.Width != 100 {
		t.Errorf("Width = %d, want 100", result.Width)
	}
	if result.Height != 200 {
		t.Errorf("Height = %d, want 200", result.Height)
	}
}

func BenchmarkProcessor_Resize(b *testing.B) {
	p := New()
	img := createTestImage(1000, 1000)
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	data := buf.Bytes()

	operations := []models.Operation{
		{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": 200, "height": 200}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(bytes.NewReader(data), "image/jpeg", operations)
	}
}

func BenchmarkProcessor_MultipleOperations(b *testing.B) {
	p := New()
	img := createTestImage(500, 500)
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	data := buf.Bytes()

	operations := []models.Operation{
		{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": 200, "height": 200}},
		{Operation: models.OperationBlur, Parameters: map[string]interface{}{"sigma": 1.0}},
		{Operation: models.OperationGrayscale},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Process(bytes.NewReader(data), "image/jpeg", operations)
	}
}
