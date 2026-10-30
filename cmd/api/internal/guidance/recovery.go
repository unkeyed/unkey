package guidance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/prompt"
)

func (f *form) recoverValidation(action cli.Action, body *cli.StringFlag, schema *Schema, values map[string]any, array, needsChanges bool) cli.Action {
	return func(ctx context.Context, cmd *cli.Command) (err error) {
		if f.screen != nil {
			defer func() { err = errors.Join(err, f.screen.finish()) }()
		}
		for {
			if f.screen != nil {
				if err := f.screen.finish(); err != nil {
					return err
				}
			}
			err := action(ctx, cmd)
			problem, ok := errors.AsType[*apierrors.BadRequestErrorResponse](err)
			if !ok || problem.Error_.Type != codes.App.Validation.InvalidInput.DocsURL() || len(problem.Error_.Errors) == 0 {
				return err
			}
			var notice strings.Builder
			first, size := utf8.DecodeRuneInString(cmd.Usage)
			fmt.Fprintf(&notice, "✗ Couldn't %c%s\n", unicode.ToLower(first), cmd.Usage[size:])
			for _, field := range problem.Error_.Errors {
				fmt.Fprintf(&notice, "  %s\n", fieldProblem(field.Location, field.Message))
			}
			var sections []section
			if f.screen != nil {
				f.screen.notice = notice.String()
				sections, f.screen.sections = f.screen.sections, nil
			} else if _, writeErr := fmt.Fprint(f.out, notice.String()); writeErr != nil {
				return writeErr
			}
			choice, promptErr := f.prompt.SelectOrdered("Edit your answers and try again?", []prompt.SelectOption{
				{Key: "cancel", Label: "Cancel"}, {Key: "edit", Label: "Edit answers"},
			})
			if promptErr != nil {
				return cancelled(promptErr, "cancelled; no request was sent")
			}
			if choice != "edit" {
				return err
			}
			if f.screen != nil {
				if err := f.screen.commit(); err != nil {
					return err
				}
				f.screen.notice, f.screen.sections = "", sections
			}
			for {
				if err := f.review(ctx, schema, values, "", needsChanges); err != nil {
					return cancelled(err, "cancelled; no request was sent")
				}
				updated, err := requestJSON(values, array)
				if err != nil {
					return err
				}
				if string(updated) == body.Value() {
					if _, err := fmt.Fprintln(f.out, "Nothing changed. Edit the rejected field first."); err != nil {
						return err
					}
					continue
				}
				if err := f.confirm(ctx, schema, values, cmd.Usage, needsChanges); err != nil {
					return cancelled(err, "cancelled; no request was sent")
				}
				updated, err = requestJSON(values, array)
				if err != nil {
					return err
				}
				if string(updated) == body.Value() {
					continue
				}
				if err := body.Parse(string(updated)); err != nil {
					return err
				}
				break
			}
		}
	}
}

// fieldProblem names the rejected field unless the API message already starts with it.
func fieldProblem(location, message string) string {
	name := label(strings.TrimPrefix(terminalSafe(location), "body."))
	message = terminalSafe(message)
	if location == "" || strings.HasPrefix(strings.ToLower(message), strings.ToLower(name)) {
		return message
	}
	return name + ": " + message
}
