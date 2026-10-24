package lazy

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Option declares the values of one label of a counter or histogram vector.
// Every label must be declared with [Values] or [Dynamic].
type Option func(*labelSpec)

// Values declares every value label can take. When all labels of a vector
// have declared values, SetRegistry creates every combination at 0.
func Values(label string, values ...string) Option {
	values = slices.Clone(values)
	return func(s *labelSpec) {
		s.values[label] = values
	}
}

// Dynamic declares that label takes values known only at runtime, such as a
// cache resource or a workspace ID. Series of a vector with a dynamic label
// are created on first use, or at 0 by [CounterVec.Bind] when the owner of
// the value is constructed.
func Dynamic(label string) Option {
	return func(s *labelSpec) {
		s.dynamic[label] = true
	}
}

type labelSpec struct {
	values  map[string][]string
	dynamic map[string]bool
}

func newLabelSpec(name string, labels []string, options []Option) labelSpec {
	s := labelSpec{values: map[string][]string{}, dynamic: map[string]bool{}}
	for _, o := range options {
		o(&s)
	}
	for l := range s.values {
		if !slices.Contains(labels, l) {
			panic(fmt.Sprintf("lazy: %s declares values for unknown label %q", name, l))
		}
		if s.dynamic[l] {
			panic(fmt.Sprintf("lazy: %s declares label %q both with values and dynamic", name, l))
		}
	}
	for l := range s.dynamic {
		if !slices.Contains(labels, l) {
			panic(fmt.Sprintf("lazy: %s declares unknown label %q dynamic", name, l))
		}
	}
	return s
}

func (s labelSpec) undeclared(labels []string) []string {
	var missing []string
	for _, l := range labels {
		if _, ok := s.values[l]; !ok && !s.dynamic[l] {
			missing = append(missing, l)
		}
	}
	return missing
}

// combinations returns every combination of declared values in label order,
// or nil if any label is dynamic.
func (s labelSpec) combinations(labels []string) [][]string {
	combos := [][]string{{}}
	for _, l := range labels {
		values, ok := s.values[l]
		if !ok {
			return nil
		}
		next := make([][]string, 0, len(combos)*len(values))
		for _, c := range combos {
			for _, v := range values {
				next = append(next, append(slices.Clone(c), v))
			}
		}
		combos = next
	}
	return combos
}

// declaration is a counter or histogram whose series SetRegistry creates.
// Gauges are not declared: their value is state the owner sets, and 0 can be
// a wrong value, for example for a timestamp.
type declaration struct {
	name       string
	undeclared []string
	prime      func()
}

func declare(name string, undeclared []string, prime func()) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.declarations = append(state.declarations, declaration{name: name, undeclared: undeclared, prime: prime})
}

func fqName(namespace, subsystem, name string) string {
	return prometheus.BuildFQName(namespace, subsystem, name)
}

// primeDeclared creates every declared series at 0. A vector label that is
// neither declared with Values nor Dynamic panics in tests, so a new metric
// without declarations fails CI; in production it only logs.
func primeDeclared() {
	state.mu.Lock()
	declarations := slices.Clone(state.declarations)
	state.mu.Unlock()

	var missing []string
	for _, d := range declarations {
		if len(d.undeclared) > 0 {
			missing = append(missing, fmt.Sprintf("%s{%s}", d.name, strings.Join(d.undeclared, ",")))
			continue
		}
		d.prime()
	}
	if len(missing) == 0 {
		return
	}

	msg := "lazy: declare these labels with lazy.Values or lazy.Dynamic: " + strings.Join(missing, ", ")
	if testing.Testing() {
		panic(msg)
	}
	logger.Warn(msg)
}
