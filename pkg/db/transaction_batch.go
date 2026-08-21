package db

import "fmt"

type transactionBatchArgument struct {
	name   string
	value  any
	result *transactionBatchResult
}

type transactionBatchResult struct {
	index int
}

func (r transactionBatchResult) reference() string {
	return fmt.Sprintf("@unkey_transaction_batch_result_%d", r.index)
}

type transactionBatchStatement struct {
	query  string
	args   []transactionBatchArgument
	result *transactionBatchResult
}

func (s transactionBatchStatement) withResultArgument(name string, result transactionBatchResult) transactionBatchStatement {
	for i := range s.args {
		if s.args[i].name == name {
			s.args[i].result = &result
			return s
		}
	}
	panic(fmt.Sprintf("transaction batch argument %q not found", name))
}
