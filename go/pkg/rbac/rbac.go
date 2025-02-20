package rbac

import (
	"fmt"
	"strings"
)

type RBAC struct{}

func New() *RBAC {
	return &RBAC{}
}

type EvaluationResult struct {
	Valid   bool
	Message string
}

func (r *RBAC) EvaluatePermissions(query PermissionQuery, permissions []string) (*EvaluationResult, error) {
	return r.evaluateQueryV1(query, permissions)
}

func (r *RBAC) evaluateQueryV1(query PermissionQuery, permissions []string) (*EvaluationResult, error) {
	// Handle simple permission check
	if query.Value != "" {
		for _, p := range permissions {
			if p == query.Value {
				return &EvaluationResult{Valid: true, Message: ""}, nil
			}
		}
		return &EvaluationResult{
			Valid:   false,
			Message: fmt.Sprintf("Missing permission: '%s'", query.Value),
		}, nil
	}

	// Handle AND operation
	if query.Operation == OperatorAnd {
		for _, child := range query.Children {
			result, err := r.evaluateQueryV1(child, permissions)
			if err != nil {
				return nil, err
			}
			if !result.Valid {
				return result, nil
			}
		}
		return &EvaluationResult{Valid: true, Message: ""}, nil
	}

	// Handle OR operation
	if query.Operation == OperatorOr {
		missingPerms := make([]string, 0)
		for _, child := range query.Children {
			result, err := r.evaluateQueryV1(child, permissions)
			if err != nil {
				return nil, err
			}
			if result.Valid {
				return result, nil
			}
			missingPerms = append(missingPerms, fmt.Sprintf("'%v'", child))
		}
		return &EvaluationResult{
			Valid: false,
			Message: fmt.Sprintf("Missing one of these permissions: [%s], have: [%s]",
				strings.Join(missingPerms, ", "),
				strings.Join(formatPermissions(permissions), ", ")),
		}, nil
	}

	return nil, fmt.Errorf("invalid query structure")
}

func formatPermissions(permissions []string) []string {
	formatted := make([]string, len(permissions))
	for i, p := range permissions {
		formatted[i] = fmt.Sprintf("'%s'", p)
	}
	return formatted
}
