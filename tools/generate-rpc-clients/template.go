package main

import (
	"embed"
	"text/template"
)

//go:embed wrapper.go.tmpl
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "wrapper.go.tmpl"))
