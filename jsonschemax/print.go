package jsonschemax

import "github.com/ory/jsonschema/v3"

func FormatError(e *jsonschema.ValidationError) (string, string) {
	var (
		err     error
		pointer string
		message string
	)

	pointer = e.InstancePtr
	message = e.Message
	switch ctx := e.Context.(type) {
	case *jsonschema.ValidationErrorContextRequired:
		if len(ctx.Missing) > 0 {
			message = "one or more required properties are missing"
			pointer = ctx.Missing[0]
		}
	}

	// We can ignore the error as it will simply echo the pointer.
	pointer, err = JSONPointerToDotNotation(pointer)
	if err != nil {
		pointer = e.InstancePtr
	}

	return pointer, message
}
