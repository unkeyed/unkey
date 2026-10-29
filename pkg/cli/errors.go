package cli

import (
	"errors"
	"fmt"
	"strings"
)

// InputErrorKind identifies caller input failures independently of their text.
type InputErrorKind string

const (
	InputErrorMissingRequiredFlag InputErrorKind = "missing_required_flag"
	InputErrorMissingFlagValue    InputErrorKind = "missing_flag_value"
	InputErrorInvalidFlagValue    InputErrorKind = "invalid_flag_value"
	InputErrorUnknownFlag         InputErrorKind = "unknown_flag"
	InputErrorUnknownCommand      InputErrorKind = "unknown_command"
	InputErrorConflictingFlags    InputErrorKind = "conflicting_flags"
)

// InputError reports invalid caller input without exposing supplied values.
// Flags contains names without dashes.
type InputError struct {
	Kind    InputErrorKind
	Flags   []string
	message string
	cause   error
}

func (e *InputError) Error() string { return e.message }

func (e *InputError) Unwrap() error { return e.cause }

func parseFlagValue(flag Flag, value string) error {
	err := flag.Parse(value)
	if err == nil {
		return nil
	}

	message := fmt.Sprintf("invalid value for flag --%s", flag.Name())
	switch flag := flag.(type) {
	case *EnumFlag:
		message += "; expected one of: " + strings.Join(flag.Allowed(), ", ")
	case *BoolFlag:
		message += "; expected a boolean"
	case *IntFlag, *Int64Flag:
		message += "; expected an integer"
	case *FloatFlag:
		message += "; expected a number"
	case *DurationFlag:
		message += "; expected a duration"
	}
	if errors.Is(err, ErrValidationFailed) && flag.Usage() != "" {
		message += "\n" + flag.Usage()
	}
	return &InputError{Kind: InputErrorInvalidFlagValue, Flags: []string{flag.Name()}, message: message, cause: err}
}
