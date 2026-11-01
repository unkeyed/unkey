package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
)

type diagnostic struct {
	Message   string                       `json:"message"`
	Status    int                          `json:"status,omitempty"`
	Type      string                       `json:"type,omitempty"`
	RequestID string                       `json:"requestId,omitempty"`
	Hint      string                       `json:"hint,omitempty"`
	Fields    []components.ValidationError `json:"fields,omitempty"`
}

func CommandError(cmd *cli.Command, path string, err error, args ...string) error {
	if err == nil {
		return nil
	}
	info := describeError(err)
	switch info.Status {
	case http.StatusUnauthorized:
		switch {
		case cmd.FlagIsSet("root-key"):
			info.Hint = "Update the --root-key credential, or remove the flag to use another credential source."
		case strings.TrimSpace(os.Getenv("UNKEY_ROOT_KEY")) != "":
			info.Hint = "Update UNKEY_ROOT_KEY, or unset it to use another credential source."
		case cmd.FlagIsSet("config"):
			info.Hint = "Update the credential in --config, or remove the flag to use a stored login."
		default:
			info.Hint = "Run 'unkey login' to sign in again, or check your saved root key and API host."
		}
	case http.StatusForbidden:
		info.Hint = "Check the credential and target workspace. See 'unkey " + path + " --help' for required access."
	case http.StatusNotFound:
		info.Hint = "Check the target and credential's workspace and access. The API does not distinguish missing resources from inaccessible ones."
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		info.Hint = "Correct the request. See 'unkey " + path + " --help' for accepted input."
		for i := range info.Fields {
			if !cmd.FlagIsSet("body") {
				field := strings.TrimPrefix(info.Fields[i].Location, "body.")
				for _, flag := range cmd.Flags {
					if strings.ReplaceAll(flag.Name(), "-", "") == strings.ToLower(field) {
						info.Fields[i].Location = "--" + flag.Name()
						break
					}
				}
			}
		}
	case http.StatusConflict, http.StatusPreconditionFailed:
		info.Hint = "Resolve the reported conflict or prerequisite before retrying. --yes does not bypass permissions or protection."
	case http.StatusTooManyRequests:
		info.Hint = "Wait before retrying. Follow any retry delay reported by the API."
	}
	query := strings.HasSuffix(path, " get") || strings.HasSuffix(path, " list") || strings.HasPrefix(path, "analytics ") || strings.HasPrefix(path, "logs ") || path == "keys whoami" || path == "workspace limits" || path == "workspace usage"
	var network net.Error
	var invalidResponse *apierrors.ResponseValidationError
	if !query && (info.Status >= 500 || errors.As(err, &network) || errors.As(err, &invalidResponse)) {
		info.Hint = "The operation may have completed. Check the resource state before retrying to avoid duplicate changes."
	}
	if errorFormat(cmd, args) == "json" {
		encoded, encodeErr := json.Marshal(struct {
			Error diagnostic `json:"error"`
		}{Error: info})
		if encodeErr != nil {
			return fmt.Errorf("encode error diagnostic: %w", encodeErr)
		}
		return errors.New(string(encoded))
	}
	_, inputError := errors.AsType[*cli.InputError](err)
	if path == "" || inputError {
		return err
	}
	var output strings.Builder
	fmt.Fprintf(&output, "Error: unkey %s failed.\n%s", path, info.Message)
	for _, field := range info.Fields {
		fmt.Fprintf(&output, "\n  %s: %s", terminalText(field.Location), terminalText(field.Message))
		if field.Fix != nil && *field.Fix != "" {
			fmt.Fprintf(&output, "\n    %s", terminalText(*field.Fix))
		}
	}
	if info.Hint != "" {
		fmt.Fprint(&output, "\n\n", info.Hint)
	}
	return errors.New(errorDiagnostic(output.String(), info.RequestID))
}

func errorFormat(cmd *cli.Command, args []string) string {
	format := cmd.Enum("error-format")
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			continue
		}
		name, value, inline := strings.Cut(strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-"), "=")
		if name == "error-format" {
			if !inline && i+1 < len(args) {
				i++
				value = args[i]
			}
			if value == "human" || value == "json" {
				format = value
			}
			continue
		}
		if !inline && flagTakesValue(cmd, name) && i+1 < len(args) {
			i++
		}
	}
	return format
}

func flagTakesValue(cmd *cli.Command, name string) bool {
	for _, flag := range cmd.Flags {
		if flag.Name() == name {
			_, boolean := flag.(*cli.BoolFlag)
			return !boolean
		}
	}
	for _, child := range cmd.Commands {
		if flagTakesValue(child, name) {
			return true
		}
	}
	return false
}

func describeError(err error) diagnostic {
	var info diagnostic
	info.Message = err.Error()
	if input, ok := errors.AsType[*cli.InputError](err); ok {
		info.Type = "urn:unkey:cli:" + string(input.Kind)
		for _, name := range input.Flags {
			info.Fields = append(info.Fields, components.ValidationError{Location: "--" + name, Message: input.Error(), Fix: nil})
		}
		return info
	}
	var response error
	for current := err; current != nil; current = errors.Unwrap(current) {
		switch current.(type) {
		case *apierrors.ForbiddenErrorResponse:
			info.Status = http.StatusForbidden
		case *apierrors.UnauthorizedErrorResponse:
			info.Status = http.StatusUnauthorized
		case *apierrors.NotFoundErrorResponse:
			info.Status = http.StatusNotFound
		case *apierrors.BadRequestErrorResponse:
			info.Status = http.StatusBadRequest
		case *apierrors.InternalServerErrorResponse:
			info.Status = http.StatusInternalServerError
		case *apierrors.ServiceUnavailableErrorResponse:
			info.Status = http.StatusServiceUnavailable
		case *apierrors.TooManyRequestsErrorResponse:
			info.Status = http.StatusTooManyRequests
		case *apierrors.ConflictErrorResponse:
			info.Status = http.StatusConflict
		case *apierrors.GoneErrorResponse:
			info.Status = http.StatusGone
		case *apierrors.PreconditionFailedErrorResponse:
			info.Status = http.StatusPreconditionFailed
		case *apierrors.UnprocessableEntityErrorResponse:
			info.Status = http.StatusUnprocessableEntity
		default:
			continue
		}
		response = current
		break
	}
	if response != nil {
		var envelope struct {
			Meta  components.Meta                   `json:"meta"`
			Error components.BadRequestErrorDetails `json:"error"`
		}
		raw, err := json.Marshal(response)
		if err != nil {
			info.Message = "Could not read the API error response."
			return info
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			info.Message = "Could not read the API error response."
			return info
		}
		info.Message = terminalText(envelope.Error.Detail)
		info.RequestID = envelope.Meta.RequestID
		info.Type = envelope.Error.Type
		info.Fields = envelope.Error.Errors
		switch info.Status {
		case http.StatusUnauthorized:
			info.Message = "Authentication failed: " + info.Message
		case http.StatusForbidden:
			info.Message = "Permission denied: " + info.Message
		case http.StatusNotFound:
			info.Message = "Not found or not accessible: " + info.Message
		case http.StatusTooManyRequests:
			info.Message = "Rate limited: " + info.Message
		case http.StatusInternalServerError:
			info.Message = "Server error: " + info.Message
		case http.StatusServiceUnavailable:
			info.Message = "Service unavailable: " + info.Message
		case http.StatusConflict:
			info.Message = "Conflict: " + info.Message
		}
		return info
	}
	if response, ok := errors.AsType[*apierrors.APIError](err); ok {
		info.Status = response.StatusCode
		info.Message = fmt.Sprintf("API request failed (HTTP %d)", response.StatusCode)
	}
	if response, ok := errors.AsType[*apierrors.ResponseValidationError](err); ok {
		info.Status = response.StatusCode
		info.Message = fmt.Sprintf("The API returned an invalid response (HTTP %d).", response.StatusCode)
	}
	var network net.Error
	if errors.As(err, &network) {
		info.Message = "Could not reach the API. Check your connection and API host."
		if network.Timeout() {
			info.Message = "The API request timed out."
		}
	}
	return info
}

func errorDiagnostic(message, requestID string) string {
	if requestID == "" {
		return message
	}
	return message + "\nRequest ID: " + terminalText(requestID)
}
