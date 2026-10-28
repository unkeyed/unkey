package spool

// FailureKind says why [Pool.Admit] did not return a [Body]. Callers map it to
// a response, for example with an exhaustive switch.
type FailureKind int

const (
	// FailureCapacity means the pool had no free slot or not enough free bytes.
	// Nothing was read from the input.
	FailureCapacity FailureKind = iota + 1
	// FailureTooLarge means the declared or streamed size exceeds
	// [Config.BodyBytesMax], or an upstream [net/http.MaxBytesReader] stopped
	// the read.
	FailureTooLarge
	// FailureInvalid means the validator rejected the body. Err wraps the
	// validator's error.
	FailureInvalid
	// FailureSourceRead means the input failed to read or close, made no
	// progress, or did not match its declared Content-Length.
	FailureSourceRead
	// FailureCanceled means the context was done before the commit point.
	FailureCanceled
	// FailureStorage means the temporary file could not be created, checked,
	// or written.
	FailureStorage
	// FailureValidatorContract means the validator accepted the body but the
	// input was not at EOF. This is a validator bug.
	FailureValidatorContract
	// FailureInternal means a caller passed an invalid argument or the pool
	// found a broken invariant. This is a programmer error.
	FailureInternal
)

// String returns a short, stable name for the kind.
func (k FailureKind) String() string {
	switch k {
	case FailureCapacity:
		return "capacity"
	case FailureTooLarge:
		return "too large"
	case FailureInvalid:
		return "invalid"
	case FailureSourceRead:
		return "source read"
	case FailureCanceled:
		return "canceled"
	case FailureStorage:
		return "storage"
	case FailureValidatorContract:
		return "validator contract"
	case FailureInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// Error is the only error type that [Pool.Admit] returns. Err holds the cause
// and any input-close, file-close, or accounting errors from cleanup.
type Error struct {
	Kind FailureKind
	Err  error
}

func (e *Error) Error() string {
	return "spool: " + e.Kind.String() + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}
