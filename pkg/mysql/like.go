package mysql

import (
	"database/sql"
	"strings"
)

// likeWildcardEscaper escapes the MySQL LIKE wildcards (% and _) and the
// default escape character (\) so user input only ever matches literally.
var likeWildcardEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ContainsPattern returns a LIKE pattern matching rows that contain search as
// a literal substring. An empty search returns an invalid NullString so
// queries guarding the filter with sqlc.narg skip it entirely.
func ContainsPattern(search string) sql.NullString {
	if search == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: "%" + likeWildcardEscaper.Replace(search) + "%", Valid: true}
}
