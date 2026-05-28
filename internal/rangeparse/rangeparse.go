package rangeparse

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func ParseUint32List(input string, min uint32, max uint32) ([]uint32, bool, error) {
	normalized := strings.TrimSpace(strings.ToLower(input))
	if normalized == "" {
		return nil, false, nil
	}
	if normalized == "all" {
		return nil, true, nil
	}

	parts := strings.Split(normalized, ",")
	seen := make(map[uint32]struct{})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false, fmt.Errorf("invalid empty token in %q", input)
		}

		if strings.Contains(part, "-") {
			if err := addRange(part, min, max, seen); err != nil {
				return nil, false, err
			}
			continue
		}

		value, err := parseBounded(part, min, max)
		if err != nil {
			return nil, false, err
		}
		seen[value] = struct{}{}
	}

	values := make([]uint32, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values, false, nil
}

func addRange(token string, min uint32, max uint32, seen map[uint32]struct{}) error {
	bounds := strings.Split(token, "-")
	if len(bounds) != 2 {
		return fmt.Errorf("invalid range token %q", token)
	}

	start, err := parseBounded(strings.TrimSpace(bounds[0]), min, max)
	if err != nil {
		return fmt.Errorf("invalid range start in %q: %w", token, err)
	}

	end, err := parseBounded(strings.TrimSpace(bounds[1]), min, max)
	if err != nil {
		return fmt.Errorf("invalid range end in %q: %w", token, err)
	}

	if start > end {
		return fmt.Errorf("invalid range %q: start greater than end", token)
	}

	for i := start; i <= end; i++ {
		seen[i] = struct{}{}
		if i == max {
			break
		}
	}

	return nil
}

func parseBounded(value string, min uint32, max uint32) (uint32, error) {
	raw, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid uint32", value)
	}

	parsed := uint32(raw)
	if parsed < min || parsed > max {
		return 0, fmt.Errorf("value %d out of bounds [%d, %d]", parsed, min, max)
	}

	return parsed, nil
}
