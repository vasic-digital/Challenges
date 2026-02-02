package bank

import (
	"encoding/json"
	"fmt"
	"os"
)

// ValidationError represents a validation issue found in a bank file.
type ValidationError struct {
	Field   string
	Message string
	Index   int // -1 if not applicable
}

func (e ValidationError) Error() string {
	if e.Index >= 0 {
		return fmt.Sprintf("challenges[%d].%s: %s", e.Index, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateFile validates a bank file structure and returns all errors found.
func ValidateFile(path string) []ValidationError {
	var errors []ValidationError

	data, err := os.ReadFile(path)
	if err != nil {
		return []ValidationError{{Field: "file", Message: err.Error(), Index: -1}}
	}

	var file BankFile
	if err := json.Unmarshal(data, &file); err != nil {
		return []ValidationError{{Field: "json", Message: err.Error(), Index: -1}}
	}

	if file.Version == "" {
		errors = append(errors, ValidationError{
			Field: "version", Message: "version is required", Index: -1,
		})
	}

	ids := make(map[string]bool)
	for i, ch := range file.Challenges {
		if ch.ID == "" {
			errors = append(errors, ValidationError{
				Field: "id", Message: "challenge ID is required", Index: i,
			})
		} else if ids[string(ch.ID)] {
			errors = append(errors, ValidationError{
				Field: "id", Message: fmt.Sprintf("duplicate ID: %s", ch.ID), Index: i,
			})
		} else {
			ids[string(ch.ID)] = true
		}

		if ch.Name == "" {
			errors = append(errors, ValidationError{
				Field: "name", Message: "challenge name is required", Index: i,
			})
		}
	}

	return errors
}
