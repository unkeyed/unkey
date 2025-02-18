package errorcode

type UnkeyDatabaseNotUniqueError struct {
	base
}

func NewUnkeyDatabaseNotUniqueError(err error) UnkeyDatabaseNotUniqueError {
	return UnkeyDatabaseNotUniqueError{
		base: newBase(
			err,
			SystemUnkey,
			NamespaceKey,
			"CONFLICT",
			"The resource identifier must be unique.",
		),
	}

}
