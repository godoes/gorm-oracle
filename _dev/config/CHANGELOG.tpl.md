{{ if .Versions -}}
{{ if .Unreleased.CommitGroups -}}

## ⭐ [最新变更]({{ .Info.RepositoryURL }}/compare/{{ $latest := index .Versions 0 }}{{ $latest.Tag.Name }}...main)

{{ range .Unreleased.CommitGroups -}}
### {{ .RawTitle }} {{ .Title }}

{{ range .Commits -}}
{{/* SKIPPING RULES - START */ -}}
{{- if not (contains .Subject " CHANGELOG") -}}
{{- if not (contains .Subject "[ci skip]") -}}
{{- if not (contains .Subject "[skip ci]") -}}
{{- if not (hasPrefix .Subject "Merge pull request ") -}}
{{- if not (hasPrefix .Subject "Merge remote-tracking ") -}}
{{- /* SKIPPING RULES - END */ -}}
- [{{ if .Type }}`{{ .Type }}`{{ end }}{{ .Subject }}]({{ $.Info.RepositoryURL }}/commit/{{ .Hash.Short }}) - `{{ datetime "2006-01-02 15:04" .Committer.Date }}`
{{- if .TrimmedBody }}
  <blockquote>

{{ indent .TrimmedBody 2 }}
  </blockquote>

{{ end -}}
{{/* SKIPPING RULES - START */ -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{/* SKIPPING RULES - END */ -}}
{{ end -}}
{{ end -}}
{{ else }}
{{- range .Unreleased.Commits -}}
{{/* SKIPPING RULES - START */ -}}
{{- if not (contains .Subject " CHANGELOG") -}}
{{- if not (contains .Subject "[ci skip]") -}}
{{- if not (contains .Subject "[skip ci]") -}}
{{- if not (hasPrefix .Subject "Merge pull request ") -}}
{{- if not (hasPrefix .Subject "Merge remote-tracking ") -}}
{{- /* SKIPPING RULES - END */ -}}
- [{{ if .Type }}`{{ .Type }}`{{ end }}{{ .Subject }}]({{ $.Info.RepositoryURL }}/commit/{{ .Hash.Short }})
{{- if .TrimmedBody }}
  <blockquote>

{{ indent .TrimmedBody 2 }}
  </blockquote>

{{ end -}}
{{/* SKIPPING RULES - START */ -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{/* SKIPPING RULES - END */ -}}
{{ end -}}
{{ end -}}
{{ end -}}

{{ range .Versions -}}
## 🔖 {{ if .Tag.Previous -}}
[`{{ .Tag.Name }}`]({{ $.Info.RepositoryURL }}/compare/{{ .Tag.Previous.Name }}...{{ .Tag.Name }})
{{- else }}`{{ .Tag.Name }}`{{ end }} - `{{ datetime "2006-01-02" .Tag.Date }}`
{{ if .CommitGroups -}}
{{ range .CommitGroups }}
### {{ .RawTitle }} {{ .Title }}

{{ range .Commits -}}
{{/* SKIPPING RULES - START */ -}}
{{- if not (contains .Subject " CHANGELOG") -}}
{{- if not (contains .Subject "[ci skip]") -}}
{{- if not (contains .Subject "[skip ci]") -}}
{{- if not (hasPrefix .Subject "Merge pull request ") -}}
{{- if not (hasPrefix .Subject "Merge remote-tracking ") -}}
{{- /* SKIPPING RULES - END */ -}}
- [{{ if .Type }}`{{ .Type }}`{{ end }}{{ .Subject }}]({{ $.Info.RepositoryURL }}/commit/{{ .Hash.Short }})
{{- if .TrimmedBody }}
  <blockquote>

{{ indent .TrimmedBody 2 }}
  </blockquote>
{{- end }}
{{/* SKIPPING RULES - START */ -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{/* SKIPPING RULES - END */ -}}
{{ end -}}
{{ end -}}
{{ else }}{{ range .Commits -}}
{{/* SKIPPING RULES - START */ -}}
{{- if not (contains .Subject " CHANGELOG") -}}
{{- if not (contains .Subject "[ci skip]") -}}
{{- if not (contains .Subject "[skip ci]") -}}
{{- if not (hasPrefix .Subject "Merge pull request ") -}}
{{- if not (hasPrefix .Subject "Merge remote-tracking ") }}
{{/* SKIPPING RULES - END */ -}}
- [{{ if .Type }}`{{ .Type }}`{{ end }}{{ .Subject }}]({{ $.Info.RepositoryURL }}/commit/{{ .Hash.Short }})
{{- if .TrimmedBody }}
  <blockquote>

{{ indent .TrimmedBody 2 }}
  </blockquote>

{{ end -}}
{{/* SKIPPING RULES - START */ -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
{{/* SKIPPING RULES - END */ -}}
{{ end -}}
{{ end -}}
{{- if .NoteGroups -}}
{{ range .NoteGroups -}}

### {{ .Title }}

{{ range .Notes }}
{{ .Body }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}
