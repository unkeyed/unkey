package main

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var caser = cases.Title(language.English)

func ToCamelCase(name string) string {
	out := ""

	for i, p := range strings.Split(name, "_") {
		if p == "id" {
			out += "ID"
		} else if p == "url" && i > 0 {
			// sqlc uses "Url" not "URL" in compound names
			out += "Url"
		} else {
			out += caser.String(p)
		}
	}

	return out
}
