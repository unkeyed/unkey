package errorpage

import (
	"bytes"
	_ "embed"
	"html/template"
)

//go:embed error.go.tmpl
var defaultTemplate string

// defaultRenderer uses the embedded HTML template.
type defaultRenderer struct {
	tmpl *template.Template
}

func (r *defaultRenderer) Render(data Data) ([]byte, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// NewRenderer returns a [Renderer] that uses the default embedded error page template.
// Panics if the template fails to parse (should never happen with an embedded template).
func NewRenderer() Renderer {
	return &defaultRenderer{
		tmpl: template.Must(template.New("error").Parse(defaultTemplate)),
	}
}
