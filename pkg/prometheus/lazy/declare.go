package lazy

import (
	"fmt"
	"slices"
	"sync"
)

// Label declares one label of a counter or histogram vector, in order.
type Label struct {
	name    string
	values  []string
	dynamic bool
}

// Values declares a label that takes a closed set of values. When every label
// of a vector is declared with Values, SetRegistry creates every combination
// at 0. Other values can still be used at runtime.
func Values(name string, values ...string) Label {
	if len(values) == 0 {
		panic(fmt.Sprintf("lazy.Values(%q): no values", name))
	}
	return Label{name: name, values: slices.Clone(values), dynamic: false}
}

// Dynamic declares a label whose values are known only at runtime, such as a
// cache resource or a workspace ID. Series of a vector with a dynamic label
// are created on first use, or at 0 by [CounterVec.Bind] when the owner of
// the value is constructed.
func Dynamic(name string) Label {
	return Label{name: name, values: nil, dynamic: true}
}

func labelNames(labels []Label) []string {
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.name
	}
	return names
}

// combinations returns every combination of declared values in label order,
// or nil if any label is dynamic.
func combinations(labels []Label) [][]string {
	combos := [][]string{{}}
	for _, l := range labels {
		if l.dynamic {
			return nil
		}
		next := make([][]string, 0, len(combos)*len(l.values))
		for _, c := range combos {
			for _, v := range l.values {
				next = append(next, append(slices.Clone(c), v))
			}
		}
		combos = next
	}
	return combos
}

// declarations are the counters and histograms whose series SetRegistry
// creates. Gauges are not declared: their value is state the owner sets, and
// 0 can be a wrong value, for example for a timestamp.
var declarations struct {
	mu     sync.Mutex
	primes []func()
}

func declare(prime func()) {
	declarations.mu.Lock()
	defer declarations.mu.Unlock()
	declarations.primes = append(declarations.primes, prime)
}

func primeDeclared() {
	declarations.mu.Lock()
	primes := slices.Clone(declarations.primes)
	declarations.mu.Unlock()

	for _, prime := range primes {
		prime()
	}
}
