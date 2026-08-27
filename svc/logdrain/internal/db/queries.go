package db

type Queries struct{ db DBTX }

func NewQueries(db DBTX) *Queries          { return &Queries{db: db} }
func (q *Queries) WithTx(tx DBTx) *Queries { return &Queries{db: tx} }
