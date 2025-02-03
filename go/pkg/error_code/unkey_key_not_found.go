package errorcode

type UnkeyKeyNotFoundError struct {
	base
}

func NewUnkeyKeyNotFoundError(err error) UnkeyKeyNotFoundError {
	return UnkeyKeyNotFoundError{
		base: newBase(
			err,
			SystemUnkey,
			NamespaceKey,
			"NOT_FOUND",
			"The requested key does not exist.",
		),
	}

}
