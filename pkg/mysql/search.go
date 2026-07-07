package mysql

import (
	"database/sql"
	"strings"
)

// likeWildcardEscaper escapes the MySQL LIKE wildcards (% and _) and the
// default escape character (\) so user input only ever matches literally.
var likeWildcardEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// SearchContains returns a LIKE pattern matching rows that contain search as
// a literal substring. The pattern is unanchored on both sides, so it cannot
// use an index and scans every candidate row. An empty search returns an
// invalid NullString so queries guarding the filter with sqlc.narg skip it
// entirely.
func SearchContains(search string) sql.NullString {
	return searchPattern("%", search, "%")
}

// SearchPrefix returns a LIKE pattern matching rows that start with search as
// a literal prefix. The pattern is left-anchored, so MySQL can serve it from
// an index on the column. An empty search returns an invalid NullString so
// queries guarding the filter with sqlc.narg skip it entirely.
func SearchPrefix(search string) sql.NullString {
	return searchPattern("", search, "%")
}

// SearchSuffix returns a LIKE pattern matching rows that end with search as a
// literal suffix. The pattern starts with a wildcard, so it cannot use an
// index and scans every candidate row. An empty search returns an invalid
// NullString so queries guarding the filter with sqlc.narg skip it entirely.
func SearchSuffix(search string) sql.NullString {
	return searchPattern("%", search, "")
}

// searchPattern escapes search so it only matches literally and anchors it
// with the given wildcards, treating an empty search as no filter.
func searchPattern(before, search, after string) sql.NullString {
	if search == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: before + likeWildcardEscaper.Replace(search) + after, Valid: true}
}
