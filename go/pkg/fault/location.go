package fault

import (
	"fmt"
	"runtime"
	"strings"
)

// withLocation represents an error with an associated call location.
// It's used internally to track where in the code errors occur.
type withLocation struct {
	// err is the underlying error being withLocation
	err error
	// location contains the file and line number where the error was withLocation
	location string
}

// Error implements the error interface by returning the underlying error messages
// in the order they were added to the chain, from oldest to newest.
//
// Example:
//
//	baseErr := fault.New("initial error")
//	withLocation := fault.Wrap(baseErr, fault.With })
//	fmt.Println(withLocation.Error()) // Output will contain all messages in chain
func (w *withLocation) Error() string {
	errs := []string{}
	chain := Unwind(w)

	for i := len(chain) - 1; i >= 0; i-- {
		errs = append(errs, chain[i].Message)
	}
	return strings.Join(errs, ": ")

}

// getLocation returns a string containing the file name and line number
// where it was called. It skips 3 frames in the call stack to get the
// actual location where the error was created rather than the internal
// function calls.
//
// The returned string is in the format "filename:linenumber"
//
// Example:
//
//	loc := getLocation() // might return "main.go:42"
func getLocation() string {
	pc := make([]uintptr, 1)
	runtime.Callers(3, pc)
	cf := runtime.CallersFrames(pc)
	f, _ := cf.Next()

	return fmt.Sprintf("%s:%d", f.File, f.Line)
}
