package processor

import (
	"image"

	"github.com/disintegration/imaging"
)

// blur applies a Gaussian blur to the image
func (p *Processor) blur(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	sigma := p.getFloatParam(params, "sigma", 3.0)
	return imaging.Blur(img, sigma), nil
}

// sharpen applies an unsharp mask to the image
func (p *Processor) sharpen(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	sigma := p.getFloatParam(params, "sigma", 1.0)
	return imaging.Sharpen(img, sigma), nil
}

// grayscale converts the image to grayscale
func (p *Processor) grayscale(img *image.NRGBA) (*image.NRGBA, error) {
	return imaging.Grayscale(img), nil
}

// sepia applies a sepia tone effect
func (p *Processor) sepia(img *image.NRGBA) (*image.NRGBA, error) {
	// Apply grayscale first, then adjust colors
	gray := imaging.Grayscale(img)
	// Apply sepia-like color adjustment
	adjusted := imaging.AdjustSaturation(gray, -100)
	adjusted = imaging.AdjustBrightness(adjusted, 5)
	return imaging.AdjustContrast(adjusted, 10), nil
}

// brightness adjusts the brightness of the image
func (p *Processor) brightness(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	amount := p.getFloatParam(params, "amount", 0)
	// Clamp to valid range
	if amount < -100 {
		amount = -100
	} else if amount > 100 {
		amount = 100
	}
	return imaging.AdjustBrightness(img, amount), nil
}

// contrast adjusts the contrast of the image
func (p *Processor) contrast(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	amount := p.getFloatParam(params, "amount", 0)
	// Clamp to valid range
	if amount < -100 {
		amount = -100
	} else if amount > 100 {
		amount = 100
	}
	return imaging.AdjustContrast(img, amount), nil
}

// saturation adjusts the color saturation of the image
func (p *Processor) saturation(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	amount := p.getFloatParam(params, "amount", 0)
	// Clamp to valid range
	if amount < -100 {
		amount = -100
	} else if amount > 100 {
		amount = 100
	}
	return imaging.AdjustSaturation(img, amount), nil
}

// rotate rotates the image by the specified angle
func (p *Processor) rotate(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	angle := p.getFloatParam(params, "angle", 0)
	return imaging.Rotate(img, angle, image.Transparent), nil
}

// flip flips the image horizontally or vertically
func (p *Processor) flip(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	horizontal := p.getBoolParam(params, "horizontal", true)
	if horizontal {
		return imaging.FlipH(img), nil
	}
	return imaging.FlipV(img), nil
}
