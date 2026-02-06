package timing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestParseEntry(t *testing.T) {
	t.Run("parses entry without labels", func(t *testing.T) {
		entry, err := ParseEntry("cache_get=1.25ms")
		require.NoError(t, err)
		require.Equal(t, "cache_get", entry.Name)
		require.Equal(t, 1250*time.Microsecond, entry.Duration)
		require.Empty(t, entry.Attributes)
	})

	t.Run("parses entry with labels", func(t *testing.T) {
		entry, err := ParseEntry("db_query{table=keys,status=miss}=53ms")
		require.NoError(t, err)
		require.Equal(t, "db_query", entry.Name)
		require.Equal(t, 53*time.Millisecond, entry.Duration)
		require.Equal(t, map[string]string{
			"table":  "keys",
			"status": "miss",
		}, entry.Attributes)
	})

	t.Run("parses units", func(t *testing.T) {
		entries := map[string]time.Duration{
			"metric=1ns":     1 * time.Nanosecond,
			"metric=1us":     1 * time.Microsecond,
			"metric=1ms":     1 * time.Millisecond,
			"metric=1s":      1 * time.Second,
			"metric=10ms":    10 * time.Millisecond,
			"metric=1.5ms":   1500 * time.Microsecond,
			"metric=2.25s":   2250 * time.Millisecond,
			"metric=100.5us": 100*time.Microsecond + 500*time.Nanosecond,
		}
		for input, expected := range entries {
			entry, err := ParseEntry(input)
			require.NoError(t, err)
			require.Equal(t, expected, entry.Duration)
		}
	})

	t.Run("parses labels with dots", func(t *testing.T) {
		entry, err := ParseEntry("cache_get{note=read.only}=1ms")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"note": "read.only"}, entry.Attributes)
	})

	t.Run("parses without whitespace", func(t *testing.T) {
		entry, err := ParseEntry("cache_get{status=miss}=1ms")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"status": "miss"}, entry.Attributes)
	})
}

func TestParseEntryErrors(t *testing.T) {
	tests := map[string]string{
		"missing name":                      "=1ms",
		"missing value":                     "cache_get=ms",
		"missing unit":                      "cache_get=12",
		"invalid unit":                      "cache_get=12msec",
		"empty labels":                      "cache_get{}=1ms",
		"invalid label key":                 "cache_get{1bad=x}=1ms",
		"missing label equals":              "cache_get{badx}=1ms",
		"missing closing brace":             "cache_get{status=miss=1ms",
		"label value empty":                 "cache_get{status=}=1ms",
		"label value contains whitespace":   "cache_get{status=miss here}=1ms",
		"label value contains comma":        "cache_get{status=miss,here}=1ms",
		"label value contains brace":        "cache_get{status=mi}ss}=1ms",
		"label value contains equals":       "cache_get{status=mi=ss}=1ms",
		"invalid number":                    "cache_get=1..2ms",
		"missing label after comma":         "cache_get{status=miss,}=1ms",
		"missing label after opening brace": "cache_get{=miss}=1ms",
		"label missing closing brace":       "cache_get{status=miss}=1ms=",
		"missing label closing brace":       "cache_get{status=miss=1ms",
		"label value contains newline":      "cache_get{status=miss\n}=1ms",
		"label value contains space":        "cache_get{status=miss here}=1ms",
		"entry contains spaces":             "cache_get=1ms extra",
		"value starts with decimal":         "cache_get=.5ms",
		"value ends with decimal":           "cache_get=5.ms",
		"value contains sign":               "cache_get=-1ms",
		"value contains exponent":           "cache_get=1e3ms",
		"name starts with digit":            "1cache=1ms",
		"label key invalid char":            "cache_get{status-flag=miss}=1ms",
		"unexpected trailing data":          "cache_get=1ms extra",
		"missing equals after labels":       "cache_get{status=miss}1ms",
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseEntry(input)
			require.Error(t, err)
		})
	}
}

func TestParseEntries(t *testing.T) {
	t.Run("parses multiple entries", func(t *testing.T) {
		entries, err := ParseEntries("cache_get=1ms,db_query{table=keys}=2ms")
		require.NoError(t, err)
		require.Len(t, entries, 2)
		require.Equal(t, "cache_get", entries[0].Name)
		require.Equal(t, time.Millisecond, entries[0].Duration)
		require.Equal(t, "db_query", entries[1].Name)
		require.Equal(t, 2*time.Millisecond, entries[1].Duration)
	})

	t.Run("rejects commas in label values", func(t *testing.T) {
		_, err := ParseEntries("metric{note=a,b}=1ms,other=2ms")
		require.Error(t, err)
	})

	t.Run("parses long header", func(t *testing.T) {
		entries := make([]string, 0, 200)
		for i := 0; i < 200; i++ {
			entries = append(entries, "metric"+strconv.Itoa(i)+"=1ms")
		}
		header := strings.Join(entries, ",")

		parsed, err := ParseEntries(header)
		require.NoError(t, err)
		require.Len(t, parsed, 200)
		require.Equal(t, "metric0", parsed[0].Name)
		require.Equal(t, 199, len(parsed)-1)
		require.Equal(t, "metric199", parsed[199].Name)
	})

	t.Run("rejects empty header", func(t *testing.T) {
		_, err := ParseEntries("")
		require.Error(t, err)
	})

	t.Run("rejects empty entries", func(t *testing.T) {
		_, err := ParseEntries("cache_get=1ms,,db_query=1ms")
		require.Error(t, err)
	})

	t.Run("rejects whitespace", func(t *testing.T) {
		_, err := ParseEntries("cache_get=1ms,db_query=2ms ")
		require.Error(t, err)
	})

	t.Run("rejects unmatched brace", func(t *testing.T) {
		_, err := ParseEntries("cache_get{status=miss=1ms,other=1ms")
		require.Error(t, err)
	})

	t.Run("rejects trailing comma", func(t *testing.T) {
		_, err := ParseEntries("cache_get=1ms,")
		require.Error(t, err)
	})
}

func TestRecord(t *testing.T) {
	t.Run("writes header when session exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		session := &zen.Session{}
		err := session.Init(recorder, req, 0)
		require.NoError(t, err)

		ctx := zen.WithSession(context.Background(), session)
		Record(ctx, Entry{Name: "cache_get", Duration: time.Millisecond})

		values := recorder.Header().Values(HeaderName)
		require.Len(t, values, 1)
		require.Equal(t, "cache_get=1ms", values[0])
	})

	t.Run("no session is a no-op", func(t *testing.T) {
		Record(context.Background(), Entry{Name: "cache_get", Duration: time.Millisecond})
	})

	t.Run("rejects invalid entry", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		session := &zen.Session{}
		err := session.Init(recorder, req, 0)
		require.NoError(t, err)

		ctx := zen.WithSession(context.Background(), session)
		Record(ctx, Entry{Name: "1bad", Duration: time.Millisecond})
		values := recorder.Header().Values(HeaderName)
		require.Empty(t, values)
	})
}

func TestEntryString(t *testing.T) {
	t.Run("formats without attributes", func(t *testing.T) {
		entry := Entry{Name: "cache_get", Duration: 1500 * time.Microsecond}
		require.Equal(t, "cache_get=1.5ms", entry.String())
	})

	t.Run("formats with attributes", func(t *testing.T) {
		entry := Entry{
			Name:     "cache_get",
			Duration: time.Millisecond,
			Attributes: map[string]string{
				"cache":  "ApiByID",
				"status": "miss",
			},
		}
		require.Equal(t, "cache_get{cache=ApiByID,status=miss}=1ms", entry.String())
	})
}
