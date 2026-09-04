package db

// Queries provides methods for all generated log drain SQL queries.
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
