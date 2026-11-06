package batch

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestProcessorClearsFlushedRows(t *testing.T) {
	var previous []string
	var tail []string
	var flushed [][]string
	processor := New(Config[string]{
		Name:          t.Name(),
		BatchSize:     4,
		BufferSize:    5,
		FlushInterval: time.Hour,
		Consumers:     1,
		Flush: func(_ context.Context, rows []string) error {
			flushed = append(flushed, slices.Clone(rows))
			if previous != nil {
				tail = slices.Clone(previous[1:])
			}
			previous = rows
			return nil
		},
	})
	t.Cleanup(processor.Close)
	for _, row := range []string{"first", "second", "third", "fourth", "last"} {
		processor.Buffer(row)
	}
	processor.Close()

	require.Equal(t, [][]string{{"first", "second", "third", "fourth"}, {"last"}}, flushed)
	require.Equal(t, []string{"", "", ""}, tail)
}

func TestProcessorRecordsFlushOutcome(t *testing.T) {
	name := t.Name()

	var mu sync.Mutex
	var attempts int
	processor := New(Config[string]{
		Name:          name,
		BatchSize:     1,
		BufferSize:    4,
		FlushInterval: time.Hour,
		Consumers:     1,
		Flush: func(_ context.Context, _ []string) error {
			mu.Lock()
			defer mu.Unlock()
			attempts++
			if attempts == 1 {
				return errors.New("flush failed")
			}
			return nil
		},
	})
	t.Cleanup(processor.Close)

	opsError := func() float64 {
		return counterValue(t, "unkey_batch_operations_total", map[string]string{"name": name, "trigger": "size_limit", "status": "error"})
	}
	opsSuccess := func() float64 {
		return counterValue(t, "unkey_batch_operations_total", map[string]string{"name": name, "trigger": "size_limit", "status": "success"})
	}
	itemsError := func() float64 {
		return counterValue(t, "unkey_batch_items_processed_errors_total", map[string]string{"name": name})
	}
	itemsSuccess := func() float64 {
		return counterValue(t, "unkey_batch_items_processed_total", map[string]string{"name": name})
	}

	errorOpsBefore := opsError()
	successOpsBefore := opsSuccess()
	errorItemsBefore := itemsError()
	successItemsBefore := itemsSuccess()

	processor.Buffer("fails")
	processor.Buffer("succeeds")
	processor.Close()

	require.Equal(t, float64(1), opsError()-errorOpsBefore, "a failed flush must be recorded as an error operation")
	require.Equal(t, float64(1), opsSuccess()-successOpsBefore, "a successful flush must be recorded as a successful operation")
	require.Equal(t, float64(1), itemsError()-errorItemsBefore, "a failed flush must count its items as errors")
	require.Equal(t, float64(1), itemsSuccess()-successItemsBefore, "only a successful flush may count its items as processed")
}

func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			matched := true
			for key, want := range labels {
				found := false
				for _, pair := range metric.GetLabel() {
					if pair.GetName() == key && pair.GetValue() == want {
						found = true
						break
					}
				}
				if !found {
					matched = false
					break
				}
			}
			if matched {
				return metric.GetCounter().GetValue()
			}
		}
	}

	return 0
}
