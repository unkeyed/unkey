package db

type transactionBatchStatement struct {
	query string
	args  []any
}
