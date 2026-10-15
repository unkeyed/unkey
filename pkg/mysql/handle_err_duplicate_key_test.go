package mysql

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestIsDuplicateKeyError(t *testing.T) {
	duplicate := &mysql.MySQLError{Number: errDuplicateKey, Message: "Duplicate entry"}
	deadlock := &mysql.MySQLError{Number: errDeadlock, Message: "Deadlock found"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "duplicate", err: duplicate, want: true},
		{name: "wrapped duplicate", err: fmt.Errorf("insert app: %w", duplicate), want: true},
		{name: "other mysql error", err: fmt.Errorf("insert app: %w", deadlock), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsDuplicateKeyError(tt.err))
		})
	}
}
