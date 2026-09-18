package input

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/llm-net/adb-claw/pkg/adb"
)

var wmSizeRe = regexp.MustCompile(`(\d+)x(\d+)`)

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
	rot := currentRotation(cmd)
	if rot == 1 || rot == 3 {
		return h, w, nil
	}
	return w, h, nil
}

func currentRotation(cmd adb.Commander) int {
	result, err := cmd.Shell("dumpsys", "window", "displays")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "mCurrentRotation=") {
			continue
		}
		for _, word := range strings.Fields(line) {
			if strings.HasPrefix(word, "mCurrentRotation=") {
				val := strings.TrimPrefix(word, "mCurrentRotation=")
				val = strings.TrimRight(val, ",")
				if r, err := strconv.Atoi(val); err == nil {
					return r & 3
				}
			}
		}
	}
	return 0
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
