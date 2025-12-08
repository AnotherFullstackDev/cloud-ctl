package lib

import (
	"fmt"
	"log/slog"
)

func ConfigEntryToTypedSlice[T any](entry interface{}, identifier string) ([]T, error) {
	l := slog.With("context", "config_entry_to_typed_slice", "identifier", identifier)

	interfaceSlice, ok := entry.([]any)
	if !ok {
		l.Debug("wrong type for entry",
			"required_type", fmt.Sprintf("%T", *new([]T)),
			"type", fmt.Sprintf("%T", entry))
		return nil, fmt.Errorf("entry must be a list of %T", *new(T))
	}

	result := make([]T, 0, len(interfaceSlice))
	for _, sliceElm := range interfaceSlice {
		l.Debug("processing slice element", "element_type", fmt.Sprintf("%T", sliceElm), "element_value", sliceElm)
		if sliceElm == nil {
			continue
		}

		elm, ok := sliceElm.(T)
		if !ok {
			l.Debug("wrong type for entry's element",
				"required_type", fmt.Sprintf("%T", *new(T)),
				"type", fmt.Sprintf("%T", sliceElm))
			return nil, fmt.Errorf("must be a list of %T", *new(T))
		}
		result = append(result, elm)
	}

	return result, nil
}
