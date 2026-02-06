package logger

import "io"

// memberlistWriter is an io.Writer that silences memberlist's internal logging.
// memberlist requires an io.Writer for its LogOutput config. We discard the
// output because memberlist is very chatty and we handle errors at the
// application level.
type memberlistWriter struct{}

// NewMemberlistWriter returns an io.Writer that discards all output.
// Use this for memberlist.Config.LogOutput to silence internal logs.
func NewMemberlistWriter() io.Writer {
	return &memberlistWriter{}
}

func (w *memberlistWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
