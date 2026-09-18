package coord

import "fmt"

// Grid is the Gemini Computer Use normalized coordinate range (inclusive).
const Grid = 999

// ToPixel maps a 0–999 normalized coordinate onto a device axis of size pixels.
func ToPixel(n, size int) int {
	if size <= 0 {
		return 0
	}
	if n < 0 {
		n = 0
	}
	if n > Grid {
		n = Grid
	}
	if size == 1 {
		return 0
	}
	return n * (size - 1) / Grid
}

// FromPixel maps a device-pixel coordinate back onto the 0–999 grid.
func FromPixel(p, size int) int {
	if size <= 1 {
		return 0
	}
	if p < 0 {
		p = 0
	}
	if p > size-1 {
		p = size - 1
	}
	return p * Grid / (size - 1)
}

// Point is a device-pixel coordinate.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Normalize maps device pixels to the 0–999 grid using current screen size.
func Normalize(x, y, width, height int) Point {
	return Point{X: FromPixel(x, width), Y: FromPixel(y, height)}
}

// Denormalize maps 0–999 grid coordinates to device pixels.
func Denormalize(nx, ny, width, height int) Point {
	return Point{X: ToPixel(nx, width), Y: ToPixel(ny, height)}
}

// ValidateNormalized returns an error if n is outside 0–999.
func ValidateNormalized(name string, n int) error {
	if n < 0 || n > Grid {
		return fmt.Errorf("%s %d is outside the 0-%d normalized grid", name, n, Grid)
	}
	return nil
}
