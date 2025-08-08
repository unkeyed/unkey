package cli

// Template for parent commands (commands with subcommands)
const parentCommandTemplate = `---
{{- if .FrontMatter }}
title: "{{ .FrontMatter.Title }}"
description: "{{ .FrontMatter.Description }}"
{{- else }}
title: "{{ .Name | title }}"
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
unkey {{ .CommandPath }}{{- if hasItems .Flags }} [flags]{{- end }}
` + "```" + `

## Quick Reference
{{- range .Subcommands }}
- ` + "`unkey {{ $.CommandPath }} {{ .Name }}`" + ` - {{ .Usage }}
{{- end }}

{{- if hasItems .Flags }}
{{- $hasRequired := false }}
{{- range .Flags }}{{- if .Required }}{{- $hasRequired = true }}{{- end }}{{- end }}
{{- if $hasRequired }}

<Banner type="warn">
Some flags are required for this command to work properly.
</Banner>

{{- end }}

## Global Flags

These flags apply to all subcommands:

{{- range .Flags }}

<Callout type="info" title="--{{ .Name }}{{- if .Required }} (required){{ end }}">
{{ .Description }}

- **Type:** {{ .Type }}
{{- if .Default }}
- **Default:** ` + "`{{ .Default }}`" + `
{{- end }}
{{- if .EnvVar }}
- **Environment:** ` + "`{{ .EnvVar }}`" + `
{{- end }}
</Callout>

{{- end }}
{{- end }}`

// Template for leaf commands (commands without subcommands)
const leafCommandTemplate = `---
{{- if .FrontMatter }}
title: "{{ .FrontMatter.Title }}"
description: "{{ .FrontMatter.Description }}"
{{- else }}
title: "{{ .Name | title }}"
description: "{{ .Usage }}"
{{- end }}
---

{{- if and .Description (ne .Description .Usage) }}
{{ .Description }}
{{- else if and .Description (eq .Description .Usage) }}
{{- /* Skip printing description if it's the same as usage */}}
{{- else if .Usage }}
{{ .Usage }}
{{- end }}

## Command Syntax

` + "```bash" + `
unkey {{ .CommandPath }}{{- if hasItems .Flags }} [flags]{{- end }}
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
{{- $hasRequired := false }}
{{- range .Flags }}{{- if .Required }}{{- $hasRequired = true }}{{- end }}{{- end }}
{{- if $hasRequired }}

<Banner type="warn">
Some flags are required for this command to work properly.
</Banner>

{{- end }}

## Flags

{{- range .Flags }}

<Callout type="info" title="--{{ .Name }}{{- if .Required }} (required){{ end }}">
{{ .Description }}

- **Type:** {{ .Type }}
{{- if .Default }}
- **Default:** ` + "`{{ .Default }}`" + `
{{- end }}
{{- if .EnvVar }}
- **Environment:** ` + "`{{ .EnvVar }}`" + `
{{- end }}
</Callout>

{{- end }}
{{- end }}
`
