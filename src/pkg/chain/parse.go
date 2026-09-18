package chain

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse turns argv groups such as `tap 180 720 tap 420 310` into specs.
func Parse(args []string) ([]Spec, error) {
	var specs []Spec
	rest := args
	for len(rest) > 0 {
		spec, n, err := parseOne(rest)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
		rest = rest[n:]
	}
	if len(specs) < MinSteps {
		return nil, fmt.Errorf("chain requires at least %d steps, got %d; use tap for a single action", MinSteps, len(specs))
	}
	if len(specs) > MaxSteps {
		return nil, fmt.Errorf("chain supports at most %d steps, got %d", MaxSteps, len(specs))
	}
	return specs, nil
}

func parseOne(args []string) (Spec, int, error) {
	if len(args) == 0 {
		return Spec{}, 0, fmt.Errorf("expected an action")
	}
	kind, err := parseKind(args[0])
	if err != nil {
		return Spec{}, 0, err
	}
	need := coordCount(kind)
	if len(args) < 1+need {
		return Spec{}, 0, fmt.Errorf("%s requires %d coordinates", kind, need)
	}
	nums, err := parseInts(args[1 : 1+need])
	if err != nil {
		return Spec{}, 0, err
	}
	spec := Spec{Kind: kind, X: nums[0], Y: nums[1]}
	if kind == KindSwipe {
		spec.X2 = nums[2]
		spec.Y2 = nums[3]
	}
	return spec, 1 + need, nil
}

func parseKind(raw string) (Kind, error) {
	switch strings.ToLower(strings.ReplaceAll(raw, "_", "-")) {
	case "tap":
		return KindTap, nil
	case "long-press":
		return KindLongPress, nil
	case "swipe":
		return KindSwipe, nil
	default:
		return "", fmt.Errorf("unknown chain action %q (tap, long-press, swipe)", raw)
	}
}

func coordCount(kind Kind) int {
	if kind == KindSwipe {
		return 4
	}
	return 2
}

func parseInts(parts []string) ([]int, error) {
	out := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid coordinate %q", part)
		}
		out[i] = n
	}
	return out, nil
}
