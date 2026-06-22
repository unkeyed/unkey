package tui

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStylesDisabledWithoutColor(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	require.Equal(t, "hello", r.Bold("hello"))
	require.Equal(t, "hello", r.Dim("hello"))
	require.Equal(t, "hello", r.Red("hello"))
	require.Equal(t, "hello", r.Green("hello"))
	require.Equal(t, "hello", r.Yellow("hello"))
	require.Equal(t, "hello", r.Cyan("hello"))
}

func TestStylesEnabledWithColor(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, true)

	require.Equal(t, "\033[1mhello\033[0m", r.Bold("hello"))
	require.Equal(t, "\033[90mhello\033[0m", r.Dim("hello"))
	require.Equal(t, "\033[32mhello\033[0m", r.Green("hello"))
}

func TestStyleEmptyStringStaysEmpty(t *testing.T) {
	r := NewWithColor(&bytes.Buffer{}, true)
	require.Equal(t, "", r.Bold(""))
}

func TestVisibleWidth(t *testing.T) {
	require.Equal(t, 5, visibleWidth("hello"))
	require.Equal(t, 5, visibleWidth("\033[32mhello\033[0m"))
	require.Equal(t, 0, visibleWidth("\033[0m"))
	require.Equal(t, 6, visibleWidth("héllo!")) // multibyte runes count once
}

func TestTableAlignsColumns(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	r.Table("ID", "STATUS").
		Row("cus_short", "ready").
		Row("cus_much_longer_id", "advancing").
		Print()

	require.Equal(t,
		"ID                  STATUS\n"+
			"cus_short           ready\n"+
			"cus_much_longer_id  advancing\n",
		buf.String())
}

func TestTableAlignsStyledCells(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, true)

	r.Table().
		Row(r.Green("ok"), "a").
		Row("long", "b").
		Print()

	require.Equal(t,
		"\033[32mok\033[0m    a\n"+
			"long  b\n",
		buf.String())
}

func TestTableIndentAndNoTrailingWhitespace(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	r.Table("A", "B").
		Indent(2).
		Row("x", "y").
		Row("xx", "").
		Print()

	require.Equal(t,
		"  A   B\n"+
			"  x   y\n"+
			"  xx  \n",
		buf.String())
}

func TestTableWithoutRowsPrintsNothing(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	r.Table("A", "B").Print()

	require.Empty(t, buf.String())
}

func TestTableRowsWiderThanHeader(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	r.Table("A").
		Row("x", "extra").
		Print()

	require.Equal(t, "A\nx  extra\n", buf.String())
}

func TestKVAlignsValues(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	r.KV().Indent(2).
		Add("status", "ready").
		Add("frozen at", "2026-06-10T19:52:51Z").
		Print()

	require.Equal(t,
		"  status     ready\n"+
			"  frozen at  2026-06-10T19:52:51Z\n",
		buf.String())
}

func TestKVAddIfSkipsEmptyValues(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	r.KV().
		Add("present", "yes").
		AddIf("absent", "").
		Print()

	require.Equal(t, "present  yes\n", buf.String())
}

func TestKVEmptyPrintsNothing(t *testing.T) {
	buf := &bytes.Buffer{}
	r := NewWithColor(buf, false)

	r.KV().Print()

	require.Empty(t, buf.String())
}
