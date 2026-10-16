package mysql

const errDuplicateKey = 1062

// IsDuplicateKeyError reports whether err is MySQL error 1062
// (duplicate key / unique constraint violation).
func IsDuplicateKeyError(err error) bool {
	return isMySQLError(err, errDuplicateKey)
}
