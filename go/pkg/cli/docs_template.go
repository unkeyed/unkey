package cli

// Template for parent commands (commands with subcommands)
const parentCommandTemplate = `---
{{- if .FrontMatter }}
title: "{{ .FrontMatter.Title }}"
description: "{{ .FrontMatter.Description }}"
{{- else }}
title: "{{ .Name | title }} Command"
description: "{{ .Usage }}"
{{- end }}
---
{{- if .Description }}
{{ .Description }}
{{- else }}
{{ .Usage }}
{{- end }}

## Quick Reference

{{- range .Subcommands }}
- ` + "`{{ $.Name }} {{ .Name }}`" + ` - {{ .Usage }}
{{- end }}

{{- if hasItems .Flags }}

## Global Flags

These flags apply to all subcommands:

| Flag | Description | Type | Default | Environment |
| --- | --- | --- | --- | --- |
{{- range .Flags }}
| ` + "`--{{ .Name }}`" + `{{- if .Required }} **(required)**{{ end }} | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} | {{- if .EnvVar }}` + "`{{ .EnvVar }}`" + `{{- else }}-{{- end }} |
{{- end }}

{{- end }}`

// Template for leaf commands (commands without subcommands)
const leafCommandTemplate = `---
{{- if .FrontMatter }}
title: "{{ .FrontMatter.Title }}"
description: "{{ .FrontMatter.Description }}"
{{- else }}
title: "{{ .Name | title }} Command"
description: "{{ .Usage }}"
{{- end }}
---

{{- if .Description }}
{{ .Description }}
{{- else }}
{{ .Usage }}
{{- end }}

## Command Syntax

` + "```bash" + `
{{ .Name }} [flags]
` + "```" + `

{{- if hasItems .Examples }}

## Examples

{{- range .Examples }}

### {{ .Title }}

` + "```bash" + `
{{ .Command }}
` + "```" + `
{{- end }}

{{- end }}

{{- if hasItems .Flags }}

## Flags

| Flag | Description | Type | Default | Environment |
| --- | --- | --- | --- | --- |
{{- range .Flags }}
| ` + "`--{{ .Name }}`" + `{{- if .Required }} **(required)**{{ end }} | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} | {{- if .EnvVar }}` + "`{{ .EnvVar }}`" + `{{- else }}-{{- end }} |
{{- end }}

{{- end }}

{{- if hasItems .EnvVars }}

## Environment Variables

| Variable | Description | Type |
| --- | --- | --- |
{{- range .EnvVars }}
| ` + "`{{ .Name }}`" + ` | {{ .Description }} | {{ .Type }} |
{{- end }}

{{- end }}`
