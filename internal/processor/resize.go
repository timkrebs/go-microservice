package processor

import (
	"image"

	"github.com/disintegration/imaging"
)

// resize scales the image to the specified dimensions
func (p *Processor) resize(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	width := p.getIntParam(params, "width", 0)
	height := p.getIntParam(params, "height", 0)

	// If both are 0, return the original
	if width == 0 && height == 0 {
		return img, nil
	}

	// Use Lanczos resampling for high quality
	return imaging.Resize(img, width, height, imaging.Lanczos), nil
}

// thumbnail creates a square thumbnail of the specified size
func (p *Processor) thumbnail(img *image.NRGBA, params map[string]interface{}) (*image.NRGBA, error) {
	size := p.getIntParam(params, "size", 150)

	// Crop and resize to a square thumbnail
	return imaging.Thumbnail(img, size, size, imaging.Lanczos), nil
}
