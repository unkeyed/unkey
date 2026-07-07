package mysql

import (
	"database/sql"
	"strings"
)

// likeEscaper escapes the MySQL LIKE wildcards (% and _) and the default
// escape character (\) so user input only ever matches literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// LikeContains builds a LIKE argument matching rows that contain search as a
// literal substring. An empty search returns an invalid NullString so queries
// guarding the filter with sqlc.narg skip it entirely.
func LikeContains(search string) sql.NullString {
	if search == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: "%" + likeEscaper.Replace(search) + "%", Valid: true}
}
