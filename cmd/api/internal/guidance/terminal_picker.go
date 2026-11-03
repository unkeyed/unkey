package guidance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
	"github.com/unkeyed/unkey/pkg/prompt"
	"golang.org/x/term"
)

func (f *terminalForm) loadResource(ctx context.Context, label string, load func(context.Context) (resourcePage, error)) (page resourcePage, err error) {
	if err := f.size(); err != nil {
		return resourcePage{}, err
	}
	if err := f.draw([]string{
		f.palette.mark("?") + " " + f.palette.bold(label),
		"  " + f.palette.hint("Loading resources…"),
		"  " + f.palette.hint(cancelHint),
	}, -1, -1); err != nil {
		return resourcePage{}, err
	}
	defer func() { err = errors.Join(err, f.discardResourceInput()) }()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		page resourcePage
		err  error
	}
	loaded := make(chan result, 1)
	go func() {
		page, err := load(ctx)
		loaded <- result{page: page, err: err}
	}()
	for {
		if err := f.loadingInput(); err != nil {
			return resourcePage{}, err
		}
		select {
		case <-ctx.Done():
			return resourcePage{}, ctx.Err()
		case result := <-loaded:
			return result.page, result.err
		default:
		}
	}
}

func (f *terminalForm) discardResourceInput() error {
	_, err := f.in.Discard(f.in.Buffered())
	return errors.Join(err, flushTerminalInput(f.fd))
}

func (f *terminalForm) Resource(label string, options []prompt.SelectOption, searchable bool, current string, schema *Schema) (value string, err error) {
	state, err := term.MakeRaw(f.fd)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, term.Restore(f.fd, state)) }()
	defer f.pending.Reset()
	input := ""
	problem := ""
	selected := 0
	f.page = 0
	filterable := false
	for i, option := range options {
		filterable = filterable || strings.HasPrefix(option.Key, "id:")
		if current != "" && option.Key == "id:"+current {
			selected = i
		}
	}
	for {
		if err := f.size(); err != nil {
			return "", err
		}
		choices := []prompt.SelectOption{}
		resources := 0
		for _, option := range options {
			if strings.HasPrefix(option.Key, "id:") {
				if strings.Contains(strings.ToLower(option.Label), strings.ToLower(input)) {
					choices = append(choices, option)
					resources++
				}
			} else if input == "" {
				choices = append(choices, option)
			}
		}
		if text := strings.TrimSpace(input); text != "" {
			choices = append(choices, prompt.SelectOption{Key: "manual:" + text, Label: "Use: " + text})
			if searchable {
				choices = append(choices, prompt.SelectOption{Key: "search:" + text, Label: "Search all for: " + text})
			}
		}
		selected = min(selected, max(0, len(choices)-1))
		lines := []string{f.palette.mark("?") + " " + f.palette.bold(label)}
		shown := input
		if shown == "" {
			shown = "Enter a value"
			if filterable {
				shown = "Filter results or enter a value"
				if visibleWidth(shown) > f.width-3 {
					shown = "Filter or enter a value"
				}
			}
			shown = f.palette.hint(shown)
		}
		lines = append(lines, "  "+shown)
		if problem != "" {
			lines = append(lines, "  "+f.palette.problem(problem))
		}
		visible := min(len(choices), max(1, f.height/2-4))
		top := min(max(0, selected-visible+1), len(choices)-visible)
		for i := top; i < top+visible; i++ {
			text := resourceRow(choices[i], f.width-3)
			if i == selected {
				lines = append(lines, f.palette.mark("❯")+" "+f.palette.value(text))
			} else {
				lines = append(lines, "  "+text)
			}
		}
		if len(choices) == 0 {
			lines = append(lines, "  "+f.palette.hint("No resources found. Enter a value."))
		}
		hints := []string{}
		if len(options) != 1 || options[0].Key != "browse" {
			count := quantity(resources, "resource", "resources")
			if top < resources && visible < resources {
				count = fmt.Sprintf("%d–%d of %s", top+1, min(top+visible, resources), count)
			}
			hints = append(hints, count)
		}
		hints = append(hints, navigationHint)
		lines = append(lines, f.footer(hints...)...)
		if err := f.draw(lines, 1, 2+uniseg.StringWidth(input)); err != nil {
			return "", err
		}
		key, err := f.key()
		if err != nil {
			return "", err
		}
		problem = ""
		switch key {
		case "\x1b":
			if input == "" {
				return "", errBack
			}
			input, selected = "", 0
		case "\r", "\n":
			if len(choices) > 0 {
				if text, manual := strings.CutPrefix(choices[selected].Key, "manual:"); manual {
					if _, err := scalarValue(schema, text); err != nil {
						problem = validationMessage(err)
						continue
					}
				}
				return choices[selected].Key, nil
			}
		case "\x1b[A", "\x10":
			selected = (selected + max(1, len(choices)) - 1) % max(1, len(choices))
		case "\x1b[B", "\x0e":
			selected = (selected + 1) % max(1, len(choices))
		case "\x7f", "\b":
			graphemes := uniseg.NewGraphemes(input)
			last := 0
			for graphemes.Next() {
				last, _ = graphemes.Positions()
			}
			input, selected = input[:last], 0
		case "\x15":
			input, selected = "", 0
		case "\x1b[5~":
			f.page = max(0, f.page-1)
		case "\x1b[6~":
			f.page++
		default:
			if !strings.ContainsFunc(key, func(r rune) bool { return !unicode.IsPrint(r) }) {
				input += key
				selected = 0
			}
		}
	}
}

func resourceRow(option prompt.SelectOption, width int) string {
	if visibleWidth(option.Label) <= width || !strings.HasPrefix(option.Key, "id:") {
		return option.Label
	}
	id := terminalSafe(strings.TrimPrefix(option.Key, "id:"))
	suffix := " (" + id + ")"
	if visibleWidth(suffix)+1 >= width {
		return truncateMiddle(id, width)
	}
	name := strings.TrimSuffix(option.Label, suffix)
	return truncateMiddle(name, width-visibleWidth(suffix)) + suffix
}

func truncateMiddle(text string, width int) string {
	if visibleWidth(text) <= width {
		return text
	}
	prefix, _ := splitColumns(text, max(0, (width-1)/2))
	suffixWidth := max(0, width-visibleWidth(prefix)-1)
	var clusters []string
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		clusters = append(clusters, graphemes.Str())
	}
	suffix := ""
	for i := len(clusters) - 1; i >= 0; i-- {
		if visibleWidth(clusters[i])+visibleWidth(suffix) > suffixWidth {
			break
		}
		suffix = clusters[i] + suffix
	}
	return prefix + "…" + suffix
}
