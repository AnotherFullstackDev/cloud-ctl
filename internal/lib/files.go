package lib

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
)

func PathMatchesOneOfPatterns(path string, patterns []string) (bool, error) {
	if path == "" {
		path = "."
	}

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		ok, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("match pattern %q: %w", pattern, err)
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}
