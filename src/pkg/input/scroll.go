package input

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/llm-net/adb-claw/pkg/adb"
)

var (
	wmSizeRe    = regexp.MustCompile(`(\d+)x(\d+)`)
	rotationRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)mCurrentRotation\s*[=:]\s*(?:ROTATION_)?(270|180|90|[0-3])`),
		regexp.MustCompile(`(?i)SurfaceOrientation\s*[=:]\s*(?:ROTATION_)?(270|180|90|[0-3])`),
		regexp.MustCompile(`(?i)mRotation\s*[=:]\s*(?:ROTATION_)?(270|180|90|[0-3])`),
		regexp.MustCompile(`(?i)\brotation\s*[=:]\s*(?:ROTATION_)?(270|180|90|[0-3])`),
	}
)

// GetScreenSize returns the current logical screen width and height.
// An override size is preferred because that is the coordinate space
// used by input and by a complete screenshot.
func GetScreenSize(cmd adb.Commander) (width, height int, err error) {
	result, err := cmd.Shell("wm", "size")
	if err != nil {
		return 0, 0, fmt.Errorf("wm size failed: %w", err)
	}
	w, h, ok := ParseWMSize(result.Stdout)
	if !ok {
		return 0, 0, fmt.Errorf("could not parse screen size from: %s", strings.TrimSpace(result.Stdout))
	}
	return w, h, nil
}

// ParseWMSize reads `wm size` output. Override size wins over physical size.
func ParseWMSize(stdout string) (width, height int, ok bool) {
	var physicalW, physicalH, overrideW, overrideH int
	for _, line := range strings.Split(stdout, "\n") {
		m := wmSizeRe.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		if w <= 0 || h <= 0 {
			continue
		}
		switch {
		case strings.Contains(line, "Override size"):
			overrideW, overrideH = w, h
		case strings.Contains(line, "Physical size"):
			physicalW, physicalH = w, h
		default:
			if physicalW == 0 {
				physicalW, physicalH = w, h
			}
		}
	}
	if overrideW > 0 && overrideH > 0 {
		return overrideW, overrideH, true
	}
	if physicalW > 0 && physicalH > 0 {
		return physicalW, physicalH, true
	}
	return 0, 0, false
}

// CurrentScreenSize returns width/height in the current rotation.
func CurrentScreenSize(cmd adb.Commander) (width, height int, err error) {
	w, h, err := GetScreenSize(cmd)
	if err != nil {
		return 0, 0, err
	}
	rot, rotErr := CurrentRotation(cmd)
	if rotErr == nil && (rot == 1 || rot == 3) {
		return h, w, nil
	}
	return w, h, nil
}

// CurrentRotation returns the current Surface rotation (0-3). OEM Android
// builds expose it under several field names and sometimes as degrees.
func CurrentRotation(cmd adb.Commander) (int, error) {
	queries := [][]string{
		{"dumpsys", "window", "displays"},
		{"dumpsys", "input"},
	}
	for _, query := range queries {
		result, err := cmd.Shell(query...)
		if err != nil {
			continue
		}
		if rotation, ok := ParseRotation(result.Stdout); ok {
			return rotation, nil
		}
	}
	return 0, fmt.Errorf("could not determine current display rotation")
}

// ParseRotation accepts common AOSP and OEM dumpsys rotation formats.
func ParseRotation(stdout string) (rotation int, ok bool) {
	var valueText string
	for _, re := range rotationRes {
		match := re.FindStringSubmatch(stdout)
		if len(match) == 2 {
			valueText = match[1]
			break
		}
	}
	if valueText == "" {
		return 0, false
	}
	value, err := strconv.Atoi(valueText)
	if err != nil {
		return 0, false
	}
	switch value {
	case 90:
		return 1, true
	case 180:
		return 2, true
	case 270:
		return 3, true
	default:
		return value & 3, true
	}
}

// ScrollDirection calculates swipe coordinates for a scroll direction.
// Returns (x1, y1, x2, y2) for the swipe.
func ScrollDirection(screenW, screenH, distance int, direction string) (x1, y1, x2, y2 int, err error) {
	centerX := screenW / 2
	centerY := screenH / 2

	switch strings.ToLower(direction) {
	case "down":
		// "scroll down" = see content below = swipe bottom→top (finger drags up)
		y1 = centerY + distance/2
		y2 = centerY - distance/2
		x1, x2 = centerX, centerX
	case "up":
		// "scroll up" = see content above = swipe top→bottom (finger drags down)
		y1 = centerY - distance/2
		y2 = centerY + distance/2
		x1, x2 = centerX, centerX
	case "right":
		// "scroll right" = see content to the right = swipe right→left (finger drags left)
		x1 = centerX + distance/2
		x2 = centerX - distance/2
		y1, y2 = centerY, centerY
	case "left":
		// "scroll left" = see content to the left = swipe left→right (finger drags right)
		x1 = centerX - distance/2
		x2 = centerX + distance/2
		y1, y2 = centerY, centerY
	default:
		return 0, 0, 0, 0, fmt.Errorf("invalid direction %q, use: up, down, left, right", direction)
	}
	return x1, y1, x2, y2, nil
}

// ScrollInBounds calculates swipe coordinates within element bounds.
func ScrollInBounds(left, top, right, bottom, distance int, direction string) (x1, y1, x2, y2 int, err error) {
	centerX := (left + right) / 2
	centerY := (top + bottom) / 2
	boundsH := bottom - top
	boundsW := right - left

	if distance == 0 {
		// Default to 60% of the element dimension
		switch strings.ToLower(direction) {
		case "up", "down":
			distance = boundsH * 60 / 100
		case "left", "right":
			distance = boundsW * 60 / 100
		}
	}

	// Pass centerX*2, centerY*2 as "virtual screen dimensions" so that
	// ScrollDirection computes center = (centerX*2)/2 = centerX, centering the swipe on the element.
	return ScrollDirection(centerX*2, centerY*2, distance, direction)
}
