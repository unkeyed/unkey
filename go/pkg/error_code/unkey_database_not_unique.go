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
			"NOT_UNIQUE",
			"The resource identifier msut be unique.",
		),
	}

}
