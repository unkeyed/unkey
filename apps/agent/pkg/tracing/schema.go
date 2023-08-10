package tracing

import "fmt"

func NewSpanName(pkg string, method string) string {
	return fmt.Sprintf("unkey.%s.%s", pkg, method)
}
