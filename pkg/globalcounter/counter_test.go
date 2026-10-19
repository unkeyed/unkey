package globalcounter_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter(t *testing.T) {
	database, _ := newDatabase(t)

	t.Run("New requires a database and a clock", func(t *testing.T) {
		counter, err := globalcounter.New(globalcounter.Config{DB: nil, Clock: clock.NewTestClock()})
		require.Error(t, err)
		require.Nil(t, counter)

		counter, err = globalcounter.New(globalcounter.Config{DB: database, Clock: nil})
		require.Error(t, err)
		require.Nil(t, counter)
	})

	t.Run("empty names are accepted", func(t *testing.T) {
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)

		require.NoError(t, counter.Add(globalcounter.AddParams{Name: "", Delta: 3}))
		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: ""})
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
	})

	t.Run("rejected overflow changes nothing locally or in MySQL", func(t *testing.T) {
		writer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
		require.NoError(t, err)
		name := uid.New(uid.RatelimitNamespacePrefix)
		added, overflowed := addsTotal(t, "ok"), addsTotal(t, "overflow")

		require.NoError(t, writer.Add(globalcounter.AddParams{Name: name, Delta: math.MaxInt64}))
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: name, Delta: -1}))
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: name, Delta: 1}))
		require.Error(t, writer.Add(globalcounter.AddParams{Name: name, Delta: 1}))
		require.Equal(t, 3.0, addsTotal(t, "ok")-added)
		require.Equal(t, 1.0, addsTotal(t, "overflow")-overflowed)
		count, err := writer.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, int64(math.MaxInt64), count)

		writer.Close()
		observer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
		require.NoError(t, err)
		t.Cleanup(observer.Close)
		count, err = observer.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, int64(math.MaxInt64), count)
	})
}
