package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func JsonUnmarshalFile(path string, i any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", path, err)
	}

	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file '%s': %w", path, err)
	}

	if err := json.Unmarshal(b, i); err != nil {
		return fmt.Errorf("failed to unmarshal '%s': %w", path, err)
	}

	return nil
}
