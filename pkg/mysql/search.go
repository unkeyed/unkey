package mysql

import (
	"database/sql"
	"strings"
)

// likeWildcardEscaper escapes the MySQL LIKE wildcards (% and _) and the
// default escape character (\) so user input only ever matches literally.
var likeWildcardEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// SearchContains returns a LIKE pattern matching rows that contain search as
// a literal substring. An empty search returns an invalid NullString so
// queries guarding the filter with sqlc.narg skip it entirely.
func SearchContains(search string) sql.NullString {
	return searchPattern("%", search, "%")
}

// SearchPrefix returns a LIKE pattern matching rows that start with search as
// a literal prefix. An empty search returns an invalid NullString so queries
// guarding the filter with sqlc.narg skip it entirely.
func SearchPrefix(search string) sql.NullString {
	return searchPattern("", search, "%")
}

// SearchSuffix returns a LIKE pattern matching rows that end with search as a
// literal suffix. An empty search returns an invalid NullString so queries
// guarding the filter with sqlc.narg skip it entirely.
func SearchSuffix(search string) sql.NullString {
	return searchPattern("%", search, "")
}

func searchPattern(before, search, after string) sql.NullString {
	if search == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: before + likeWildcardEscaper.Replace(search) + after, Valid: true}
}
