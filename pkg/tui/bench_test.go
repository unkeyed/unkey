package tui

import (
	"io"
	"testing"
)

func BenchmarkTablePrint(b *testing.B) {
	r := NewWithColor(io.Discard, true)
	for b.Loop() {
		t := r.Table("ID", "STATUS", "PERIOD ENDS")
		for range 50 {
			t.Row("cus_UgEeZp1mjNpcYy", r.Green("ready"), "2026-07-01T19:52:51Z")
		}
		t.Print()
	}
}

func BenchmarkKVPrint(b *testing.B) {
	r := NewWithColor(io.Discard, true)
	for b.Loop() {
		r.KV().Indent(2).
			Add("status", r.Green("ready")).
			Add("frozen at", "2026-06-10T19:52:51Z").
			Add("customer", "cus_UgEeZp1mjNpcYy").
			Print()
	}
}

func BenchmarkVisibleWidth(b *testing.B) {
	s := "\033[32mcus_UgEeZp1mjNpcYy\033[0m"
	for b.Loop() {
		visibleWidth(s)
	}
}
