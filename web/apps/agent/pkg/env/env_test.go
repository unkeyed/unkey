package env_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/env"
)

func TestString_WhenSet(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	value := uuid.NewString()

	t.Setenv(key, value)

	got := e.String(key)
	require.Equal(t, got, value)
}

func TestString_WhenNotSet(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.Error(t, err) },
	}

	key := uuid.NewString()

	got := e.String(key)
	require.Equal(t, "", got)
}

func TestString_WhenNotSetFallback(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	fallback := uuid.NewString()

	got := e.String(key, fallback)
	require.Equal(t, fallback, got)
}

func TestStringsAppend_WhenSet(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	values := []string{uuid.NewString(), uuid.NewString()}

	t.Setenv(key, strings.Join(values, ","))

	got := e.StringsAppend(key)
	require.Equal(t, got, values)
}

func TestStringsAppend_WhenSetWithDefaults(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	values := []string{uuid.NewString(), uuid.NewString()}
	defaults := []string{uuid.NewString(), uuid.NewString()}

	t.Setenv(key, strings.Join(values, ","))

	got := e.StringsAppend(key, defaults)
	require.Equal(t, 4, len(got))
	require.Contains(t, got, values[0])
	require.Contains(t, got, values[1])
	require.Contains(t, got, defaults[0])
	require.Contains(t, got, defaults[1])
}

func TestStringsAppend_WhenNotSet(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.Error(t, err) },
	}

	key := uuid.NewString()

	got := e.StringsAppend(key)
	require.Equal(t, []string{}, got)
}

func TestStringsAppend_WhenNotSetFallback(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	fallback := []string{uuid.NewString()}

	got := e.StringsAppend(key, fallback)
	require.Equal(t, fallback, got)
}

func TestInt_WhenSet(t *testing.T) {

	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	value := int(rand.NewSource(time.Now().UnixNano()).Int63())

	t.Setenv(key, fmt.Sprintf("%d", value))

	got := e.Int(key)
	require.Equal(t, got, value)
}

func TestInt_WhenNotSet(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.Error(t, err) },
	}

	key := uuid.NewString()

	got := e.Int(key)
	require.Equal(t, 0, got)
}

func TestInt_WhenNotSetFallback(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	fallback := int(rand.NewSource(time.Now().UnixNano()).Int63())

	got := e.Int(key, fallback)
	require.Equal(t, fallback, got)
}

func TestBool_WhenSet(t *testing.T) {

	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	value := true

	t.Setenv(key, fmt.Sprintf("%t", value))

	got := e.Bool(key)
	require.Equal(t, got, value)
}

func TestBool_WhenNotSet(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.Error(t, err) },
	}

	key := uuid.NewString()

	got := e.Bool(key)
	require.Equal(t, false, got)
}

func TestBool_WhenNotSetFallback(t *testing.T) {
	e := env.Env{
		ErrorHandler: func(err error) { require.NoError(t, err) },
	}

	key := uuid.NewString()
	fallback := true

	got := e.Bool(key, fallback)
	require.Equal(t, fallback, got)
}
