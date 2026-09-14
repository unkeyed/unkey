package main

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	generator := NewGenerator()
	require.NoError(t, generator.Configure([]byte(`{"package":"db"}`)))

	response, err := generator.Generate(&plugin.GenerateRequest{Queries: []*plugin.Query{
		{
			Name:     "UpdateUser",
			Cmd:      ":exec",
			Comments: []string{" transactional-batch-statement"},
			Params: []*plugin.Parameter{
				{Column: &plugin.Column{Name: "name"}},
				{Column: &plugin.Column{Name: "id"}},
			},
		},
		{
			Name:     "InsertAudit",
			Cmd:      ":exec",
			Comments: []string{"transactional-batch-statement"},
			Params: []*plugin.Parameter{
				{Column: &plugin.Column{Name: "event_id"}},
				{Column: &plugin.Column{Name: "payload"}},
			},
		},
	}})
	require.NoError(t, err)
	require.Len(t, response.GetFiles(), 1)
	file := response.GetFiles()[0]
	require.Equal(t, "transaction_batch_statements_generated.go", file.GetName())
	content := string(file.GetContents())
	require.Contains(t, content, "func updateUserTransactionBatchStatement(params UpdateUserParams) transactionBatchStatement")
	require.Contains(t, content, "query: updateUser")
	require.Contains(t, content, "params.Name")
	require.Contains(t, content, "params.ID")
	require.Contains(t, content, "func insertAuditTransactionBatchStatement(params InsertAuditParams) transactionBatchStatement")
	require.Contains(t, content, "query: insertAudit")
	require.Contains(t, content, "params.EventID")
	require.Contains(t, content, "params.Payload")
}

func TestGenerateRejectsDuplicateDirective(t *testing.T) {
	t.Parallel()

	generator := NewGenerator()

	_, err := generator.Generate(&plugin.GenerateRequest{Queries: []*plugin.Query{{
		Name:     "UpdateUser",
		Cmd:      ":exec",
		Comments: []string{batchStatementDirective, " " + batchStatementDirective},
	}}})
	require.ErrorContains(t, err, "duplicate transactional-batch-statement directives")
}

func TestGenerateRejectsRowQuery(t *testing.T) {
	t.Parallel()

	generator := NewGenerator()

	_, err := generator.Generate(&plugin.GenerateRequest{Queries: []*plugin.Query{{
		Name:     "FindUser",
		Cmd:      ":one",
		Comments: []string{batchStatementDirective},
	}}})
	require.ErrorContains(t, err, "must use :exec")
}
