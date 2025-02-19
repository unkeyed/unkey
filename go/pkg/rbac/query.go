// query.go
package rbac

type QueryOperator string

const (
	OperatorNil QueryOperator = ""
	OperatorAnd QueryOperator = "and"
	OperatorOr  QueryOperator = "or"
)

type PermissionQuery struct {
	Operation QueryOperator     `json:"operation,omitempty"`
	Value     string            `json:"value,omitempty"`
	Children  []PermissionQuery `json:"children,omitempty"`
}

func And(queries ...PermissionQuery) PermissionQuery {
	return PermissionQuery{
		Operation: OperatorAnd,
		Value:     "",
		Children:  queries,
	}
}

func Or(queries ...PermissionQuery) PermissionQuery {
	return PermissionQuery{
		Operation: OperatorOr,
		Value:     "",
		Children:  queries,
	}
}

func P(permission string) PermissionQuery {
	return PermissionQuery{
		Operation: OperatorNil,
		Value:     permission,
		Children:  []PermissionQuery{},
	}
}
