package errorsext

import "fmt"

type ValidationErrors []ValidationError

func (ve *ValidationErrors) Error() string {
	messages := make([]string, 0, len(*ve))
	for _, err := range *ve {
		messages = append(messages, err.Error())
	}

	return fmt.Sprintf("validation errors: %s", messages)
}

func (ve *ValidationErrors) AddError(field, message string) {
	if *ve == nil {
		*ve = make(ValidationErrors, 0)
	}

	*ve = append(*ve, ValidationError{Field: field, Message: message})
}

func (ve *ValidationErrors) HasErrors() bool {
	return ve != nil && len(*ve) > 0
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("field: %s, message: %s", ve.Field, ve.Message)
}
