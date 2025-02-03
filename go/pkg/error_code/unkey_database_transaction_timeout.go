package errorcode

type UnkeyDatabaseTransactionTimeoutError struct {
	base
}

func NewUnkeyDatabaseTransactionTimeoutError(err error) UnkeyDatabaseTransactionTimeoutError {
	return UnkeyDatabaseTransactionTimeoutError{
		base: newBase(
			err,
			SystemUnkey,
			NamespaceDatabase,
			"TRANSACTION_TIMEOUT",
			"The transaction reached its max timeout and aborted.",
		),
	}

}
