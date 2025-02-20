package otel

import "fmt"

func NewSpanName(pkg string, method string) string {
	return fmt.Sprintf("%s.%s", pkg, method)
}
