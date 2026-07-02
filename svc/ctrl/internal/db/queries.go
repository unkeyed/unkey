package db

// Queries provides methods for all generated control plane SQL queries.
type Queries struct {
	db DBTX
}

// NewQueries binds generated query methods to db.
func NewQueries(db DBTX) *Queries {
	return &Queries{db: db}
}

// WithTx binds generated query methods to tx.
func (q *Queries) WithTx(tx DBTx) *Queries {
	return &Queries{db: tx}
}

// BulkQueries provides methods for generated bulk insert queries.
type BulkQueries struct {
	db DBTX
}

// NewBulkQueries binds generated bulk query methods to db.
func NewBulkQueries(db DBTX) *BulkQueries {
	return &BulkQueries{db: db}
}

// WithTx binds generated bulk query methods to tx.
func (q *BulkQueries) WithTx(tx DBTx) *BulkQueries {
	return &BulkQueries{db: tx}
}
