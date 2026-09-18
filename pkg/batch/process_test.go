package batch

import (
	"context"
	"slices"
	"testing"
	"time"

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
		Flush: func(_ context.Context, rows []string) {
			flushed = append(flushed, slices.Clone(rows))
			if previous != nil {
				tail = slices.Clone(previous[1:])
			}
			previous = rows
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
